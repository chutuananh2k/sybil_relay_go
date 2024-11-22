package core

import (
	"context"
	"io"
	"log"
	"net"
	"time"
)

func proxy(ctx context.Context, dst, src net.Conn) func() error {
	return func() error {
		defer func() {
			log.Println("proxy exited")
		}()
		for {
			// checking that sister goroutine is still alive
			select {
			case <-ctx.Done():
				// sister goroutine signals cancel
				log.Println("sister goroutine signals cancel")
				return nil
			default:
				// good to continue
			}
			n, err := io.Copy(dst, src)
			if err != nil {
				log.Println("proxy copy error:", err)
				return err
			}

			// without this check, the loop will consume all the cpu once either dest or src close
			if n <= 0 {
				log.Println("proxy end, nothing more to copy")
				return nil
			}

			// reset deadline
			// todo config deadline
			dst.SetDeadline(time.Now().Add(10 * time.Second))
			src.SetDeadline(time.Now().Add(10 * time.Second))
			//dst.SetDeadline(time.Now().Add(30*time.Minute))
		}
	}
}
