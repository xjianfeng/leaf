package timer_test

import (
	"fmt"
	"server/leaf/timer"
	"testing"
	"time"
)

func GoTime() {
	d := timer.NewDispatcher(10)
	// timer 1
	d.AfterFunc(5, func() {
		fmt.Println("My name is Leaf")
	})

	// timer 2
	t := d.AfterFunc(1, func() {
		fmt.Println("will not print")
	})
	t.Stop()

	// dispatch
	(<-d.ChanTimer).Cb()

}

func ExampleCronExpr() {
	cronExpr, err := timer.NewCronExpr("0 * * * *")
	if err != nil {
		return
	}

	fmt.Println(cronExpr.Next(time.Date(
		2000, 1, 1,
		20, 10, 5,
		0, time.UTC,
	)))

	// Output:
	// 2000-01-01 21:00:00 +0000 UTC
}

func ExampleCron() {
	d := timer.NewDispatcher(10)

	// cron expr
	println(time.Now().Unix())
	cronExpr, err := timer.NewCronExpr("* * * * * *")
	if err != nil {
		return
	}

	println(time.Now().Unix())
	// cron
	//var c *timer.Cron
	//c = d.CronFunc(cronExpr, func() {
	d.CronFunc(cronExpr, func() {
		fmt.Println("My name is Leaf")
		//c.Stop()
	})

	// dispatch
	(<-d.ChanTimer).Cb()

	// Output:
	// My name is Leaf
}

func TestGoTime(t *testing.T) {
	fmt.Println("................................")
	ExampleCron()
}
