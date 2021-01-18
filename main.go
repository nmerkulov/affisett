package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	signal "os/signal"
	"sync"
	"syscall"
	"time"
)

type payload struct {
	URLs []string `json:"urls"`
}

type response struct {
	URLs []string `json:"urls"`
}

func readURL(ctx context.Context, c *http.Client, u string) (string, error) {
	handleErr := func(err error) (string, error) {
		return "", fmt.Errorf("readURL: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return handleErr(err)
	}
	resp, err := c.Do(req)
	if err != nil {
		return handleErr(err)
	}
	defer resp.Body.Close()
	res, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return handleErr(err)
	}

	if len(res) > 50 {
		res = res[:50]
	}
	return string(res), nil
}

func main() {
	rateLimit := make(chan struct{}, 100)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		select {
		case rateLimit <- struct{}{}:
			defer func() {
				<-rateLimit
			}()
		default:
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		p := payload{}
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if len(p.URLs) > 20 {
			http.Error(w, http.StatusText(http.StatusUnprocessableEntity), http.StatusUnprocessableEntity)
			return
		}
		var (
			wg          sync.WaitGroup
			urls        = make([]string, len(p.URLs))
			sem         = make(chan struct{}, 4)
			firstErr    error
			errOnce     sync.Once
			ctx, cancel = context.WithCancel(r.Context())
			client      = &http.Client{
				Timeout: time.Second * 1,
			}
		)
		defer cancel()
	L:
		for i, u := range p.URLs {
			select {
			case <-ctx.Done():
				break L
			case sem <- struct{}{}:
				wg.Add(1)
				go func(i int, u string) {
					defer func() {
						<-sem
						wg.Done()
					}()
					r, err := readURL(ctx, client, u)
					if err != nil {
						errOnce.Do(func() {
							firstErr = err
							cancel()
						})
						return
					}
					urls[i] = r
				}(i, u)
			}
		}
		wg.Wait()
		if firstErr != nil {
			http.Error(w, firstErr.Error(), http.StatusFailedDependency)
			return
		}
		if err := json.NewEncoder(w).Encode(response{URLs: urls}); err != nil {
			log.Println(fmt.Errorf("handler: %w", err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	})

	srv := &http.Server{
		Addr: ":8080",
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}
