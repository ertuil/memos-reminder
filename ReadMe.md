# Memos-reminder

A SMTP reminder for `usememos/memos`. It reads the memos from the `memos`' webhook, and 1) sends reminder E-mails when date/time is detected in it; 2) provide a '.ics' calender.

Some usecases (written in the memos' markdown content):

```
A meeting this afternoon @2024-01-20 14:00@

Weekly meetings @2024-01-18 14:00/1w@

Every mornings @08:00/1d@
```


## Install

Build the docker image:
```
docker build -t memos-reminder .
```

Then, complete the config files, an example is in `example/config.yaml`.

```
mkdir data
touch data/config.yaml
```

And run the docker images:

```
docker-compose up -d
```

## Usage

Finally, you should set the the webhook url in the `memos` settings. The URL is `http(s)://<your_ip>:8880/reminder/webhook`

Also, you can download the `.ics` calender file at: `http(s)://<your_ip>:8880/reminder/cal?uid=<uid>&pw=<ics_pw>`