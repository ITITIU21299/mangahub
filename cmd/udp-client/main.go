package main

import (
	"flag"
	"fmt"
	"log"

	"mangahub/internal/user"
)

// Simple UDP client for UC-009/UC-010:
// - register mode: sends a register message and listens for a confirmation
// - notify mode: sends a chapter_release notification (admin trigger)

type registerMessage struct {
	Type        string   `json:"type"`
	UserID      string   `json:"user_id"`
	MangaIDs    []string `json:"manga_ids,omitempty"`
	Preferences []string `json:"preferences,omitempty"`
	ClientLabel string   `json:"client_label,omitempty"`
}

type registerResponse struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type notification struct {
	Type      string `json:"type"`
	MangaID   string `json:"manga_id"`
	Title     string `json:"title"`
	Chapter   int    `json:"chapter"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

func main() {
	mode := flag.String("mode", "register", "mode: register | notify")
	addr := flag.String("addr", "localhost:9091", "UDP server address")
	userID := flag.String("user", "", "user ID for registration")
	mangaID := flag.String("manga", "", "manga ID for notification")
	title := flag.String("title", "", "manga title for notification")
	chapter := flag.Int("chapter", 1, "chapter number for notification")
	message := flag.String("message", "New chapter released!", "notification message")
	flag.Parse()

	switch *mode {
	case "register":
		if *userID == "" {
			log.Fatal("register mode requires -user flag")
		}
		if err := doRegister(*addr, *userID); err != nil {
			log.Fatal("register error:", err)
		}
	case "notify":
		if *mangaID == "" || *title == "" {
			log.Fatal("notify mode requires -manga and -title flags")
		}
		if err := doNotify(*addr, *mangaID, *title, *chapter, *message); err != nil {
			log.Fatal("notify error:", err)
		}
	default:
		log.Fatalf("unknown mode: %s", *mode)
	}
}

func doRegister(addr, userID string) error {
	msg, err := user.RegisterForUDPNotifications(user.UDPRegisterOptions{
		ServerAddr:  addr,
		UserID:      userID,
		ClientLabel: "cli-client",
	})
	if err != nil {
		return err
	}
	fmt.Printf("Registration response: %s\n", msg)
	return nil
}

func doNotify(addr, mangaID, title string, chapter int, msg string) error {
	if err := user.SendUDPNotification(user.UDPNotification{
		ServerAddr: addr,
		MangaID:    mangaID,
		Title:      title,
		Chapter:    chapter,
		Message:    msg,
	}); err != nil {
		return err
	}
	fmt.Println("Notification sent.")
	return nil
}


