package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type WebhookMemoT struct {
	Content      string        `json:"content"`
	CreatedTs    int64         `json:"createdTs"`
	CreatorId    int           `json:"creatorId"`
	MemoId       int           `json:"id"`
	Pinned       bool          `json:"pinned"`
	RelationList []interface{} `json:"relationList"`
	ResourceList []interface{} `json:"resourceList"`
	UpdatedTs    int64         `json:"updatedTs"`
	Visibility   string        `json:"visibility"`
}

type WebhookT struct {
	ActivityType string       `json:"activityType"`
	CreatedTs    int64        `json:"createdTs"`
	CreatorId    int          `json:"creatorId"`
	Memo         WebhookMemoT `json:"memo"`
	URL          string       `json:"url"`
}

type WebhookResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func HTTPServe(ctx context.Context) {

	srv := http.Server{
		Addr:         Config.Listen,
		WriteTimeout: 30 * time.Second,
	}

	http.HandleFunc("/reminder/webhook", HandleWebhook)
	slog.Info("HTTP Server start", "listen", Config.Listen)

	defer srv.Shutdown(ctx)

	go func() {
		<-ctx.Done()
		srv.Shutdown(ctx)
	}()

	err := srv.ListenAndServe()
	if err != nil {
		slog.Info("HTTP Server shutdown", "error", err)
	}
}

func HandleWebhook(w http.ResponseWriter, r *http.Request) {
	slog.Info("Got webhook", "method", r.Method, "remote", r.RemoteAddr, "url", r.URL, "type", r.Header.Get("Content-Type"))

	var resp = WebhookResponse{
		Code:    500,
		Message: "error",
	}

	resp_b, _ := json.Marshal(resp)

	if r.Method != "POST" {
		slog.Error("Invalid method", "method", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write(resp_b)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Failed to read body", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(resp_b)
		return
	}

	var webhook_content WebhookT
	err = json.Unmarshal(body, &webhook_content)
	if err != nil {
		slog.Error("Failed to parse body", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(resp_b)
		return
	}
	slog.Debug("Got webhook content", "content", webhook_content, "activityType", webhook_content.ActivityType)

	activityType := webhook_content.ActivityType

	if activityType == "memos.memo.deleted" {
		err = HandleDeletedMemo(webhook_content)
	} else if activityType == "memos.memo.created" || activityType == "memos.memo.updated" {
		err = ParseWebhook(webhook_content)
	} else {
		err = errors.New("unsupported activityType")
	}

	// err = ParseWebhook(webhook_content)
	if err != nil {
		slog.Error("Failed to parse webhook", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(resp_b)
		return
	}

	resp = WebhookResponse{
		Code:    0,
		Message: "ok",
	}

	resp_b, _ = json.Marshal(resp)
	w.WriteHeader(http.StatusOK)
	w.Write(resp_b)
}

func ParseWebhook(body WebhookT) (err error) {
	mid := body.Memo.MemoId
	uid := body.CreatorId

	cts, nextTss, diff_sec, err := ParseContent(body.Memo.Content)
	if err != nil {
		return
	}

	slog.Debug("ParseWebhook", "cts", cts, "nextTss", nextTss, "diff_sec", diff_sec, "mid", mid, "uid", uid)

	db.Model(&TimerDB{}).Where("memo_id = ? AND user = ?", mid, uid).Delete(&TimerDB{})

	for idx, ct := range cts {
		nextTs := nextTss[idx]
		diffSec := diff_sec[idx]
		newTimer := TimerDB{
			User:     uid,
			MemoId:   mid,
			Content:  ct,
			NextTs:   nextTs,
			Diff_sec: diffSec,
		}
		db.Create(&newTimer)
	}

	LoadTimerFromDB()

	return nil
}

func ParseContent(content string) (cts []string, NextTss []time.Time, DiffSecs []int, err error) {
	lines := strings.Split(content, "\n")
	currentTs := time.Now()

	for _, line := range lines {
		t1 := strings.Index(line, "@")
		if t1 == -1 {
			continue
		}
		if t1+1 >= len(line) {
			continue
		}
		t2 := strings.Index(line[t1+1:], "@")
		if t2 == -1 {
			continue
		}

		reminder_str := line[t1+1 : t1+t2+1] // 这是日期字符串

		var diff_div int = 0

		time_part := strings.Split(reminder_str, "/")
		if len(time_part) >= 2 {
			// 计算间隔（秒数）
			diff_str := time_part[1]
			diff_factor := 1
			if strings.HasSuffix(diff_str, "m") {
				diff_factor = 60
				diff_str = diff_str[:len(diff_str)-1]
			} else if strings.HasSuffix(diff_str, "h") {
				diff_factor = 3600
				diff_str = diff_str[:len(diff_str)-1]
			} else if strings.HasSuffix(diff_str, "d") {
				diff_factor = 86400
				diff_str = diff_str[:len(diff_str)-1]
			} else if strings.HasSuffix(diff_str, "w") {
				diff_factor = 604800
				diff_str = diff_str[:len(diff_str)-1]
			} else if strings.HasSuffix(diff_str, "M") {
				diff_factor = 2592000
				diff_str = diff_str[:len(diff_str)-1]
			} else if strings.HasSuffix(diff_str, "y") {
				diff_factor = 31536000
				diff_str = diff_str[:len(diff_str)-1]
			}

			diff_base, err := strconv.Atoi(diff_str)
			if err != nil {
				continue
			}
			if diff_base == 0 {
				diff_div = -1
			} else {
				diff_div = diff_base * diff_factor
			}
		} else {
			diff_div = -1
		}

		time_part[0] = strings.TrimSuffix(time_part[0], "/")
		date_time_part := strings.Split(time_part[0], " ")
		date_time_str := time_part[0]
		if len(date_time_part) == 1 {
			date_time_str = currentTs.Format("2006-01-02") + " " + date_time_part[0]
			if diff_div == 0 {
				// 只有日期，默认设置每天循环
				diff_div = 86400
			}
		}

		if diff_div < 0 {
			diff_div = 0
		}

		baseTs, err := time.ParseInLocation("2006-01-02 15:04", date_time_str, time.Local)
		slog.Debug("debug", "ct", currentTs, "bts", baseTs, "error", err)

		if err != nil {
			continue
		}

		if baseTs.Before(currentTs) {
			if diff_div == 0 {
				continue
			}
			// 平移到下一个提醒时刻
			diff_sec := int(currentTs.Sub(baseTs).Seconds())
			move_times := diff_sec/diff_div + 1
			baseTs = baseTs.Add(time.Duration(move_times*diff_div) * time.Second)
		}

		slog.Debug("Recognize time", "nextTs", baseTs, "diff_sec", diff_div)
		cts = append(cts, line)
		NextTss = append(NextTss, baseTs)
		DiffSecs = append(DiffSecs, diff_div)
	}
	return cts, NextTss, DiffSecs, nil
}

func HandleDeletedMemo(body WebhookT) (err error) {
	uid := body.CreatorId
	mid := body.Memo.MemoId

	slog.Debug("Delete memos", "uid", uid, "mid", mid)
	db.Model(&TimerDB{}).Where("memo_id = ? AND user = ?", mid, uid).Delete(&TimerDB{})
	LoadTimerFromDB()

	return nil
}
