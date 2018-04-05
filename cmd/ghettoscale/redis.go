package main

import (
	"time"

	"github.com/garyburd/redigo/redis"
)

// NewRedisPoolFromURL returns a new *redigo/redis.Pool configured for th1e supplied url
// The url can include a password in the standard form and if so is used to AUTH against
// the redis server.
func NewRedisPoolFromURL(url string) (*redis.Pool, error) {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.DialURL(url)
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}, nil
}
