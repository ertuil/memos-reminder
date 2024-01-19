package main

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/arran4/golang-ical"
)

func ParseICS(uid int) string {
	var timer_list []TimerDB
	db.Model(&TimerDB{}).Where("user = ?", uid).Find(&timer_list)

	cal := ics.NewCalendar()
	cal.SetMethod(ics.MethodRequest)
	for _, timer := range timer_list {
		server_id := fmt.Sprintf("%d@memo-reminders", timer.User)
		event := cal.AddEvent(fmt.Sprintf("%s-%d-%d", server_id, timer.MemoId, timer.ID))
		event.SetCreatedTime(timer.CreatedAt)
		event.SetDtStampTime(timer.CreatedAt)
		event.SetModifiedAt(timer.UpdatedAt)
		event.SetStartAt(timer.NextTs)
		event.SetEndAt(timer.NextTs.Add(30 * time.Minute))

		body := "[Memos 定时提醒]" +  " 用户ID: "+ strconv.Itoa(timer.User) + " 备忘编号：" + strconv.Itoa(timer.MemoId) + " 内容：" + timer.Content
		event.SetSummary(timer.Content)
		event.SetDescription(body)
		if timer.Diff_sec%31536000 == 0 {
			intv := timer.Diff_sec / 31536000
			if intv == 1 {
				event.AddRrule(fmt.Sprintf("FREQ=YEARLY;BYMONTH=%d;BYMONTHDAY=%d", timer.NextTs.Month(), timer.NextTs.Day()))
			} else {
				event.AddRrule(fmt.Sprintf("FREQ=YEARLY;INTERVAL=%d;BYMONTH=%d;BYMONTHDAY=%d", intv, timer.NextTs.Month(), timer.NextTs.Day()))
			}
		} else if timer.Diff_sec%2592000 == 0 {
			intv := timer.Diff_sec / 2592000
			if intv == 1 {
				event.AddRrule(fmt.Sprintf("FREQ=MONTHLY;BYMONTHDAY=%d", timer.NextTs.Day()))
			} else {
				event.AddRrule(fmt.Sprintf("FREQ=MONTHLY;INTERVAL=%d;BYMONTHDAY=%d", intv, timer.NextTs.Day()))
			}
		} else if timer.Diff_sec%604800 == 0 {
			intv := timer.Diff_sec / 604800
			if intv == 1 {
				event.AddRrule(fmt.Sprintf("FREQ=WEEKLY;BYDAY=%s;WKST=MO", timer.NextTs.Weekday().String()[:2]))
			} else {
				event.AddRrule(fmt.Sprintf("FREQ=WEEKLY;INTERVAL=%d;BYDAY=%s;WKST=MO", intv, timer.NextTs.Weekday().String()[:2]))
			}
		} else if timer.Diff_sec%86400 == 0 {
			intv := timer.Diff_sec / 86400
			if intv == 1 {
				event.AddRrule(fmt.Sprintf("FREQ=DAILY"))
			} else {
				event.AddRrule(fmt.Sprintf("FREQ=DAILY;INTERVAL=%d", intv))
			}
		}
		// event.AddRrule(fmt.Sprintf("FREQ=YEARLY;BYMONTH=%d;BYMONTHDAY=%d", time.Now().Month(), time.Now().Day()))
		event.SetOrganizer(server_id, ics.WithCN("memo-reminders"))
		event.AddAttendee("reciever or participant", ics.CalendarUserTypeIndividual, ics.ParticipationStatusNeedsAction, ics.ParticipationRoleReqParticipant, ics.WithRSVP(true))
	}
	cal_str := cal.Serialize()
	slog.Info("Generate calender", "cal", cal_str)
	return cal_str
}
