run(){
  out=/tmp/chromedp-detect-redirect.out
  go run main.go 2>&1 | tee $out
  echo output is in $out
}
"$@"