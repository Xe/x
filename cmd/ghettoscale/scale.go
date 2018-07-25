package main

import (
	"context"
	"fmt"
	"net/url"
)

// State is the state of the ghetto autoscaler.
type State int

/*
States that this can be in

    digraph G {
	ok [label="OK"]
	scaling_up [label="Scaling Up"]
	scaling_down [label="Scaling Down"]
	max_scale [label="Max Scale"]

	init -> ok [label="test passes"]
	init -> scaling_up [label="test fails"]
	ok -> scaling_up [label="test fails"]
	scaling_up -> scaling_up [label="test fails"]
	scaling_up -> max_scale [label="test fails"]
	scaling_up -> ok [label="test passes"]
	scaling_down -> ok [label="minimum\nscale"]
	ok -> scaling_down [label="test has\npassed\nn times"]
	scaling_down -> scaling_down [label="test passes"]
	max_scale -> scaling_down [label="test passes"]
	max_scale -> ok [label="test passes"]
    }

This is the overall state machine for the autoscaler.
*/
const (
	Init State = iota
	OK
	ScalingUp
	MaxScale
	ScalingDown
)

func Check(ctx context.Context, u *url.URL) error {
	switch u.Scheme {
	case "heroku":
		return CheckHeroku(ctx, u)
	}

	return fmt.Errorf("no such scheme for %s", u.Scheme)
}

func CheckHeroku(ctx context.Context, u *url.URL) error {
	// q := u.Query()
	var err error

	// heroku://printerfacts/web?kind=lang&metric=g:go.routines_max&min=1&max=3&stagger=1&mode=threshold&threshold=500
	// appID := u.Host
	// processType := u.Path[1:]
	// kind := q.Get("kind")
	// metric := q.Get("metric")
	// min := q.Get("min")
	// max := q.Get("max")
	// stagger := q.Get("stagger")
	// mode := q.Get("mode")

	// get redis connection
	//  check if app:process type is currently staggered in redis
	//   if so decrement stagger counter in redis and exit
	// fetch metrics data from metrics-api
	// fetch current number of dynos for this app id and process type
	// fetch state of the

	return err
}
