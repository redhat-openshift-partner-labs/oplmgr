package internal

import (
	"bytes"
	"embed"
	"github.com/gobuffalo/envy"
	mail "github.com/xhit/go-simple-mail/v2"
	"time"

	"html/template"
	"log"
)

var timezonetext = map[string]string{
  "americas": "For the Americas region this means from 9am to 5pm UTC-5",
	"apac": "For the Asia and Pacific regions this means from 9am to 5pm UTC+7",
	"emea": "For the Europe, Middle East, and Africa regions this means from 9am to 5pm UTC+1",
}

//go:embed assets/*
var assetData embed.FS

func init() {
	// Load environment variables
	err := envy.Load(".env"); if err != nil {
		log.Printf("Unable to load environment variables: %v\n", err)
	}
}

func SendWelcomeEmail(to *[]string, cc *[]string, bcc *[]string, clusterinfo map[string]string) {
	var b bytes.Buffer

	server := mail.NewSMTPClient()

	server.Port = 587
	server.Host = envy.Get("SMTP_HOST", "localhost")
	server.Username = envy.Get("SMTP_USER", "")
	server.Password = envy.Get("SMTP_PASSWORD", "")
	server.Encryption = mail.EncryptionSTARTTLS
	server.KeepAlive = false
	server.ConnectTimeout = 10 * time.Second
	server.SendTimeout = 10 * time.Second

	smtpClient, err := server.Connect()
	if err != nil {
		log.Fatalf("Unable to create client so failing: %v\n", err)
	}

	t, err := template.ParseFS(assetData, "assets/welcome.html")
	if err != nil {
		log.Printf("Unable to parse welcome email html template: %v\n", err)
	}

	welcome := struct{
		ConsoleURL string
		KubeAdminLink string
		KubeConfigLink string
		Timezone string
	}{
		ConsoleURL: clusterinfo["consoleurl"],
		KubeAdminLink: clusterinfo["kubeadmin"],
		KubeConfigLink: clusterinfo["kubeconfig"],
		Timezone: timezonetext[clusterinfo["timezone"]],
	}

	err = t.Execute(&b, &welcome); if err != nil { log.Printf("Unable to execute template: %v\n", err) }

  email := mail.NewMSG()
  email.SetFrom("OpenShift Partner Labs <opl-no-reply@redhat.com>").
  	AddTo(*to...).
  	AddCc(*cc...).
  	AddBcc(*bcc...).
  	SetSubject("OpenShift Partner Lab " + clusterinfo["clusterid"] + " - " + clusterinfo["company"])

  email.SetBody(mail.TextHTML, b.String())

  if email.Error != nil {
  	log.Fatalf("An error occurred prior to sending: %v\n", email.Error)
	}

	err = email.Send(smtpClient); if err != nil {
		log.Printf("An error occurred sending email: %v\n", err)
	} else {
		log.Println("welcome email sent successfully.")
	}
}

func SendCredsEmail(to *[]string, cc *[]string, bcc *[]string, clusterinfo map[string]string) {
	var b bytes.Buffer

	server := mail.NewSMTPClient()

	server.Port = 587
	server.Host = envy.Get("SMTP_HOST", "localhost")
	server.Username = envy.Get("SMTP_USER", "")
	server.Password = envy.Get("SMTP_PASSWORD", "")
	server.Encryption = mail.EncryptionSTARTTLS
	server.KeepAlive = false
	server.ConnectTimeout = 10 * time.Second
	server.SendTimeout = 10 * time.Second

	smtpClient, err := server.Connect()
	if err != nil {
		log.Fatalf("Unable to create client so failing: %v\n", err)
	}

	t, err := template.ParseFS(assetData, "assets/credentials.html")
	if err != nil {
		log.Printf("Unable to parse credentials email html template: %v\n", err)
	}

	credentials := struct{
		ConsoleURL string
		KubeAdminLink string
		KubeConfigLink string
	}{
		ConsoleURL: clusterinfo["consoleurl"],
		KubeAdminLink: clusterinfo["kubeadmin"],
		KubeConfigLink: clusterinfo["kubeconfig"],
	}

	err = t.Execute(&b, &credentials); if err != nil { log.Printf("Unable to execute template: %v\n", err) }

	email := mail.NewMSG()
	email.SetFrom("OpenShift Partner Labs <opl-no-reply@redhat.com>").
		AddTo(*to...).
		AddCc(*cc...).
		AddBcc(*bcc...).
		SetSubject("OpenShift Partner Lab Credentials " + clusterinfo["clusterid"] + " - " + clusterinfo["company"])

	email.SetBody(mail.TextHTML, b.String())

	if email.Error != nil {
		log.Fatalf("An error occurred prior to sending: %v\n", email.Error)
	}

	err = email.Send(smtpClient); if err != nil {
		log.Printf("An error occurred sending email: %v\n", err)
	} else {
		log.Println("credentials email sent successfully.")
	}
}

func SendAdminEmail(to *[]string, cc *[]string, bcc *[]string, clusterinfo map[string]string) {
	var b bytes.Buffer

	server := mail.NewSMTPClient()

	server.Port = 587
	server.Host = envy.Get("SMTP_HOST", "localhost")
	server.Username = envy.Get("SMTP_USER", "")
	server.Password = envy.Get("SMTP_PASSWORD", "")
	server.Encryption = mail.EncryptionSTARTTLS
	server.KeepAlive = false
	server.ConnectTimeout = 10 * time.Second
	server.SendTimeout = 10 * time.Second

	smtpClient, err := server.Connect()
	if err != nil {
		log.Fatalf("Unable to create client so failing: %v\n", err)
	}

	t, err := template.ParseFS(assetData, "assets/kubeadmin.html")
	if err != nil {
		log.Printf("Unable to parse kubeadmin email html template: %v\n", err)
	}

	kubeadmin := struct{
		ConsoleURL string
		KubeAdminLink string
		KubeConfigLink string
	}{
		ConsoleURL: clusterinfo["consoleurl"],
		KubeAdminLink: clusterinfo["kubeadmin"],
		KubeConfigLink: clusterinfo["kubeconfig"],
	}

	err = t.Execute(&b, &kubeadmin); if err != nil { log.Printf("Unable to execute template: %v\n", err) }

	email := mail.NewMSG()
	email.SetFrom("OpenShift Partner Labs <opl-no-reply@redhat.com>").
		AddTo(*to...).
		AddCc(*cc...).
		AddBcc(*bcc...).
		SetSubject("OpenShift Partner Lab Credentials " + clusterinfo["clusterid"] + " - " + clusterinfo["company"])

	email.SetBody(mail.TextHTML, b.String())

	if email.Error != nil {
		log.Fatalf("An error occurred prior to sending: %v\n", email.Error)
	}

	err = email.Send(smtpClient); if err != nil {
		log.Printf("An error occurred sending email: %v\n", err)
	} else {
		log.Println("kubeadmin email sent successfully.")
	}
}

func SendConfigEmail(to *[]string, cc *[]string, bcc *[]string, clusterinfo map[string]string) {
	var b bytes.Buffer

	server := mail.NewSMTPClient()

	server.Port = 587
	server.Host = envy.Get("SMTP_HOST", "localhost")
	server.Username = envy.Get("SMTP_USER", "")
	server.Password = envy.Get("SMTP_PASSWORD", "")
	server.Encryption = mail.EncryptionSTARTTLS
	server.KeepAlive = false
	server.ConnectTimeout = 10 * time.Second
	server.SendTimeout = 10 * time.Second

	smtpClient, err := server.Connect()
	if err != nil {
		log.Fatalf("Unable to create client so failing: %v\n", err)
	}

	t, err := template.ParseFS(assetData, "assets/kubeconfig.html")
	if err != nil {
		log.Printf("Unable to parse kubeconfig email html template: %v\n", err)
	}

	kubeconfig := struct{
		ConsoleURL string
		KubeAdminLink string
		KubeConfigLink string
	}{
		ConsoleURL: clusterinfo["consoleurl"],
		KubeAdminLink: clusterinfo["kubeadmin"],
		KubeConfigLink: clusterinfo["kubeconfig"],
	}

	err = t.Execute(&b, &kubeconfig); if err != nil { log.Printf("Unable to execute template: %v\n", err) }

	email := mail.NewMSG()
	email.SetFrom("OpenShift Partner Labs <opl-no-reply@redhat.com>").
		AddTo(*to...).
		AddCc(*cc...).
		AddBcc(*bcc...).
		SetSubject("OpenShift Partner Lab Credentials " + clusterinfo["clusterid"] + " - " + clusterinfo["company"])

	email.SetBody(mail.TextHTML, b.String())

	if email.Error != nil {
		log.Fatalf("An error occurred prior to sending: %v\n", email.Error)
	}

	err = email.Send(smtpClient); if err != nil {
		log.Printf("An error occurred sending email: %v\n", err)
	} else {
		log.Println("kubeconfig email sent successfully.")
	}
}