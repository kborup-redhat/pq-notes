package daemon

import "time"

func SendNotification(title, folder string, due time.Time) error {
	body := folder + " — Due: " + due.Format("02-01-2006 15:04")
	return sendOSNotification("pq-notes: "+title, body)
}
