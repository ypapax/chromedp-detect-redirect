package main

import (
	"context"
	"fmt"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/sirupsen/logrus"
	"log"
	"os"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	logrus.SetReportCaller(true)
	logrus.SetLevel(logrus.TraceLevel)
	myDefaultExecAllocatorOptions := [...]chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,

		// After Puppeteer's default behavior.
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("enable-features", "NetworkService,NetworkServiceInProcess"),
		chromedp.Flag("disable-background-timer-throttling", true),
		chromedp.Flag("disable-backgrounding-occluded-windows", true),
		chromedp.Flag("disable-breakpad", true),
		chromedp.Flag("disable-client-side-phishing-detection", true),
		chromedp.Flag("disable-default-apps", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-features", "site-per-process,TranslateUI,BlinkGenPropertyTrees"),
		chromedp.Flag("disable-hang-monitor", true),
		chromedp.Flag("disable-ipc-flooding-protection", true),
		chromedp.Flag("disable-popup-blocking", true),
		chromedp.Flag("disable-prompt-on-repost", true),
		chromedp.Flag("disable-renderer-backgrounding", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("force-color-profile", "srgb"),
		chromedp.Flag("metrics-recording-only", true),
		chromedp.Flag("safebrowsing-disable-auto-update", true),
		chromedp.Flag("enable-automation", true),
		chromedp.Flag("password-store", "basic"),
		chromedp.Flag("use-mock-keychain", true),
		chromedp.Flag("disable-web-security", true),
	}

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()
	var opts = myDefaultExecAllocatorOptions[:]
	chromeContext, cancelContext := chromedp.NewExecAllocator(ctx, opts...)
	defer cancelContext()
	chromeContext2, cancel3 := chromedp.NewContext(chromeContext, chromedp.WithLogf(log.Printf))
	defer cancel3()
	log.Println("starting")
	// Grab the first spawned tab that isn't blank.
	/*ch := chromedp.WaitNewTarget(chromeContext2, func(info *target.Info) bool {
		log.Printf("info: %+v\n", info)
		return info.URL != ""
	})*/
	var response, contentType string
	var responseHeaders map[string]interface{}
	var statusCode int64
	var detectRedirectUrl string
	u := os.Getenv("START_URL")
	lu := logrus.WithField("u", u)
	if err := chromedp.Run(chromeContext2,
		chromeTask(
			chromeContext2, u,
			map[string]interface{}{"User-Agent": "Mozilla/5.0"},
			&response, &statusCode, &contentType, &responseHeaders, &detectRedirectUrl),
	); err != nil {
		panic(err)
	}
	lu.Tracef("statusCode %+v, detectRedirectUrl: %+v", statusCode, detectRedirectUrl)
	/*log.Println("before waiting a chan")
	newCtx, cancel := chromedp.NewContext(chromeContext2, chromedp.WithTargetID(<-ch))
	defer cancel()

	var urlstr string
	log.Println("before getting loc")
	if err := chromedp.Run(newCtx, chromedp.Location(&urlstr)); err != nil {
		panic(err)
	}
	log.Println("new tab's path:", urlstr)*/

}

func chromeTask(chromeContext context.Context, url string, requestHeaders map[string]interface{}, response *string, statusCode *int64, contentType *string, responseHeaders *map[string]interface{}, detectRederectedUrl *string) chromedp.Tasks {
	l := logrus.WithField("url", url)
	chromedp.ListenTarget(chromeContext, func(event interface{}) {
		switch responseReceivedEvent := event.(type) {
		case *network.EventResponseReceived:
			response := responseReceivedEvent.Response
			if response.URL == url {
				*statusCode = response.Status
				*responseHeaders = response.Headers
				lb := logrus.WithField("response.URL", response.URL).WithField("statusCode", *statusCode).
					WithField("responseHeaders", responseHeaders).
					WithField("contentType", *contentType).
					WithField("url", url)
				lb.Tracef("got headers and status")
			}
			lc := l.WithField("response.URL", "response.URL: "+response.URL).
				WithField("response.Status", fmt.Sprintf("status: %+v", response.Status))
			lc.Tracef("EventResponseReceived")
		case *network.EventRequestWillBeSent:
			request := responseReceivedEvent.Request
			l.Tracef("chromedp is requesting url (could be in background): %s\n", request.URL)
			if responseReceivedEvent.RedirectResponse != nil {
				from := responseReceivedEvent.RedirectResponse.URL
				to := request.URL
				if url == from {
					l.Tracef(" got redirect: from %+v to %s", from, to)
					url = to
					if detectRederectedUrl != nil {
						*detectRederectedUrl = to
					}
				}
			}
		case *page.EventDownloadProgress:
			l.Tracef("page.EventDownloadProgress: TotalBytes: %+v, State: %+v",
				responseReceivedEvent.TotalBytes, responseReceivedEvent.State)
		}
	})
	return chromedp.Tasks{
		network.Enable(),
		network.SetExtraHTTPHeaders(requestHeaders),
		page.SetDownloadBehavior(page.SetDownloadBehaviorBehaviorDeny).WithDownloadPath("."),
		chromedp.Navigate(url),
		chromedp.ActionFunc(func(ctx context.Context) error {
			l.Tracef("ActionFunc")
			return nil
		})}
}