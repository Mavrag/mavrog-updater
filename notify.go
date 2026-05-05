package main

import (
	toast "git.sr.ht/~jackmordaunt/go-toast/v2"
)

func sendToast(title, body, activationURL string) {
	n := toast.Notification{
		AppID: "MavrogUpdater",
		Title: title,
		Body:  body,
	}
	if activationURL != "" {
		n.ActivationArguments = activationURL
	}
	_ = n.Push()
}
