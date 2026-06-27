// Command capture regenerates the README web-mode screenshots
// (docs/assets/web-*.png) by driving a headless browser against a running
// `fur serve docs/demo/`. It lives in the e2e module so chromedp (and its
// heavy, Go-1.26 browser deps) never touch the main module.
//
// It writes PNGs; the Makefile `capture-web` target starts the server,
// runs this, and converts the PNGs to webp. Browser auto-detection covers the
// standard names plus the Playwright/rod download caches, so it needs no
// system Chromium (Ubuntu 24.04 ships only a snap, which chromedp can't drive).
//
//	cd e2e && go run ./capture -url http://127.0.0.1:7777 -out ../docs/assets
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/chromedp/chromedp"
)

func findBrowser() string {
	if p := os.Getenv("FUR_CHROME"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	globs := []string{
		filepath.Join(home, ".cache/ms-playwright/chromium-*/chrome-linux64/chrome"),
		filepath.Join(home, ".cache/rod/browser/chromium-*/chrome"),
		filepath.Join(home, ".cache/puppeteer/chrome/*/chrome-linux64/chrome"),
	}
	for _, g := range globs {
		if m, _ := filepath.Glob(g); len(m) > 0 {
			return m[len(m)-1] // newest-sorted last
		}
	}
	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "chrome"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}

type shot struct {
	name string
	do   func(base string) chromedp.Tasks
}

func main() {
	base := flag.String("url", "http://127.0.0.1:7777", "base URL of a running `fur serve`")
	out := flag.String("out", "../docs/assets", "output directory for PNGs")
	w := flag.Int("w", 1280, "viewport width")
	h := flag.Int("h", 720, "viewport height")
	flag.Parse()

	browser := findBrowser()
	if browser == "" {
		log.Fatal("no Chrome/Chromium found (set FUR_CHROME or install one)")
	}
	log.Printf("browser: %s", browser)

	doc := "/millions-of-cats/millions-of-cats.md"
	shots := []shot{
		{"web-cover", func(b string) chromedp.Tasks {
			return chromedp.Tasks{
				chromedp.Navigate(b + doc),
				chromedp.WaitVisible(`article.markdown-body img`, chromedp.ByQuery),
				chromedp.Sleep(800 * time.Millisecond), // let the cover decode
			}
		}},
		{"web-reading", func(b string) chromedp.Tasks {
			return chromedp.Tasks{
				chromedp.Navigate(b + doc),
				chromedp.WaitVisible(`article.markdown-body img[src*="i_006"]`, chromedp.ByQuery),
				chromedp.ScrollIntoView(`article.markdown-body img[src*="i_006"]`, chromedp.ByQuery),
				chromedp.Sleep(800 * time.Millisecond),
			}
		}},
		{"web-graph", func(b string) chromedp.Tasks {
			return chromedp.Tasks{
				chromedp.Navigate(b + "/graph"),
				chromedp.WaitVisible(`#graph svg circle`, chromedp.ByQuery), // d3 (CDN) rendered nodes
				chromedp.Sleep(2500 * time.Millisecond),                     // force simulation settles
			}
		}},
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(browser),
		chromedp.Headless,
		chromedp.NoSandbox,
		chromedp.DisableGPU,
		chromedp.WindowSize(*w, *h),
	)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	for _, s := range shots {
		ctx, cancel := chromedp.NewContext(allocCtx)
		ctx, tcancel := context.WithTimeout(ctx, 45*time.Second)

		var buf []byte
		tasks := append(s.do(*base),
			chromedp.EmulateViewport(int64(*w), int64(*h)),
			chromedp.CaptureScreenshot(&buf),
		)
		if err := chromedp.Run(ctx, tasks); err != nil {
			tcancel()
			cancel()
			log.Fatalf("%s: %v", s.name, err)
		}
		path := filepath.Join(*out, s.name+".png")
		if err := os.WriteFile(path, buf, 0o644); err != nil {
			tcancel()
			cancel()
			log.Fatalf("write %s: %v", path, err)
		}
		fmt.Printf("  wrote %s (%d KB)\n", path, len(buf)/1024)
		tcancel()
		cancel()
	}
}
