package main

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/Andrew-M-C/go.timeconv"
)

var (
	timer_list_lock sync.RWMutex
	timer_list      *[]TimerDB = new([]TimerDB)
)

const (
	PERIOD = 10 * time.Second

	CHECK_PERIOD = 4 * time.Hour
)

func LoadTimerFromDB() (err error) {
	timer_list_lock.Lock()
	timer_list, err = LoadTimers()
	timer_list_lock.Unlock()
	return
}

func TimerServe(ctx context.Context) (err error) {
	slog.Info("Timer start")
	err = LoadTimerFromDB()
	if err != nil {
		return
	}

	ticker := time.NewTicker(PERIOD)

	check_timer := time.NewTicker(CHECK_PERIOD)
	// 一定要调用Stop()，回收资源
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("TimerServe Quit")
			return
		case <-ticker.C:
			// 每5秒中从chan t.C 中读取一次
			go handleTimers(ctx)
		case <-check_timer.C:
			go handleCheck(ctx)
		}
	}
}

func handleTimers(ctx context.Context) {
	check_st := time.Now()
	check_et := check_st.Add(PERIOD)

	slog.Debug("Check Timer", "start", check_st, "end", check_et, "list", timer_list)

	activate_flag := false
	timer_list_lock.RLock()
	for _, timer := range *timer_list {
		if timer.NextTs.After(check_st) && timer.NextTs.Before(check_et) {
			handleActivateTimer(ctx, timer)
			activate_flag = true
		}
	}
	timer_list_lock.RUnlock()

	if activate_flag {
		err := LoadTimerFromDB()
		if err != nil {
			slog.Error("Failed to load timer from database", "error", err)
		}
	}
}

func handleActivateTimer(ctx context.Context, timer TimerDB) {
	slog.Info("Activate Timer", "user", timer.User, "memo_id", timer.MemoId, "content", timer.Content, "next_ts", timer.NextTs, "rotate", timer.Diff_sec)

	go SendSMTP(timer.User, timer.MemoId, timer.NextTs, timer.Content)

	if time.Duration(timer.Diff_sec)*time.Second <= PERIOD {
		db.Delete(&timer)
		slog.Info("Delete Timer", "user", timer.User, "memo_id", timer.MemoId, "content", timer.Content, "next_ts", timer.NextTs, "rotate", timer.Diff_sec)
	} else {
		if timer.Diff_sec >= 2592000 &&  timer.Diff_sec < 31536000 && timer.Diff_sec % 2592000 == 0 {
			// 整月份调整
			c := timer.Diff_sec / 2592000
			timer.NextTs = timeconv.AddDate(timer.NextTs, 0, c, 0)
		} else if timer.Diff_sec >= 31536000 && timer.Diff_sec % 31536000 == 0 {
			// 整年调整
			c := timer.Diff_sec / 31536000
			timer.NextTs = timeconv.AddDate(timer.NextTs, c, 0, 0)
		} else {
			// 默认按秒调整
			timer.NextTs = timer.NextTs.Add(time.Duration(timer.Diff_sec) * time.Second)
		}
		db.Save(&timer)
		slog.Info("Reset Timer", "user", timer.User, "memo_id", timer.MemoId, "content", timer.Content, "next_ts", timer.NextTs, "rotate", timer.Diff_sec)
	}
}

func handleCheck(ctx context.Context) {
	slog.Info("Reload all timer from database")
	LoadTimerFromDB()
}
