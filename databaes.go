package main

import (
	"log/slog"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var (
	db *gorm.DB
)

type TimerDB struct {
	gorm.Model
	User     int       `gorm:"column:user;type:int"`
	MemoId   int       `gorm:"column:memo_id;type:int"`
	Content  string    `gorm:"column:content;type:text"`
	NextTs   time.Time `gorm:"column:next_ts;type:datetime"`
	Diff_sec int       `gorm:"column:diff_sec;type:int"`
}

func InitDatabase() (err error) {
	db, err = gorm.Open(sqlite.Open(database_file), &gorm.Config{})
	if err != nil {
		return
	}
	db.AutoMigrate(&TimerDB{})
	return nil
}

func LoadTimers() (timer_list *[]TimerDB, err error) {
	db.Find(&timer_list)
	currentTs := time.Now()
	for idx, timer := range *timer_list {
		if timer.NextTs.Before(currentTs) {
			if timer.Diff_sec == 0 {
				continue
			}

			diff_sec := int(currentTs.Sub(timer.NextTs).Seconds())
			move_times := diff_sec/timer.Diff_sec + 1
			timer.NextTs = timer.NextTs.Add(time.Duration(move_times*timer.Diff_sec) * time.Second)
			if move_times > 0 {
				db.Save(timer)
				(*timer_list)[idx] = timer
			}
		}
		slog.Info("load timer from database", "user", timer.User, "memo_id", timer.MemoId, "content", timer.Content, "next_ts", timer.NextTs, "rotate", timer.Diff_sec)
	}
	return
}
