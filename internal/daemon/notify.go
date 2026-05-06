package daemon

import "time"

func SendNotification(title, customer string, due time.Time) error {
	body := "Customer: " + customer + " — Due: " + due.Format("02-01-2006 15:04")
	return sendOSNotification("pq-notes: "+title, body)
}
