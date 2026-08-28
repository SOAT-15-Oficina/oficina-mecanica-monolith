package port

import "github.com/SOAT-15-Oficina/oficina-mecanica-monolith/packages/email"

// EmailSender is the application outbound port for email delivery.
// Implemented by packages/email (MailHog in development).
type EmailSender = email.Provider

// EmailMessage is the email payload used by the port.
type EmailMessage = email.Message
