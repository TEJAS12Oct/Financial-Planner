package mailer

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"html"
	"log/slog"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	Host      string
	Port      int
	Username  string
	Password  string
	FromEmail string
	FromName  string
	UseTLS    bool
	UseSSL    bool
}

type Mailer struct {
	Config Config
}

// SendReportData sends an in-memory PDF as an email attachment.
func (m Mailer) SendReportData(
	to string,
	customerName string,
	filename string,
	pdfData []byte,
) error {
	startedAt := time.Now()
	to = strings.TrimSpace(to)
	fromEmail := strings.TrimSpace(m.Config.FromEmail)

	slog.Info(
		"SMTP email delivery started",
		"smtp_host", m.Config.Host,
		"smtp_port", m.Config.Port,
		"from", fromEmail,
		"to", to,
		"attachment", filepath.Base(strings.TrimSpace(filename)),
		"attachment_bytes", len(pdfData),
	)

	fail := func(stage string, err error) error {
		wrapped := fmt.Errorf("%s: %w", stage, err)
		slog.Error(
			"SMTP email delivery failed",
			"stage", stage,
			"smtp_host", m.Config.Host,
			"smtp_port", m.Config.Port,
			"from", fromEmail,
			"to", to,
			"duration_ms", time.Since(startedAt).Milliseconds(),
			"error", err,
		)
		return wrapped
	}

	if err := ValidateAddress(to); err != nil {
		return fail("validate recipient email", err)
	}

	if err := ValidateAddress(m.Config.FromEmail); err != nil {
		return fail("validate sender email", err)
	}

	if strings.TrimSpace(m.Config.Host) == "" {
		return fail("validate SMTP configuration", fmt.Errorf("SMTP host is empty"))
	}

	if m.Config.Port <= 0 {
		return fail("validate SMTP configuration", fmt.Errorf("invalid SMTP port: %d", m.Config.Port))
	}

	if strings.TrimSpace(m.Config.Username) == "" {
		return fail("validate SMTP configuration", fmt.Errorf("SMTP username is empty"))
	}

	if strings.TrimSpace(m.Config.Password) == "" {
		return fail("validate SMTP configuration", fmt.Errorf("SMTP password is empty"))
	}

	if len(pdfData) == 0 {
		return fail("validate attachment", fmt.Errorf("PDF attachment data is empty"))
	}

	filename = filepath.Base(strings.TrimSpace(filename))
	if filename == "" || filename == "." {
		filename = "Financial_Planning_Report.pdf"
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	from := mail.Address{
		Name:    strings.TrimSpace(m.Config.FromName),
		Address: strings.TrimSpace(m.Config.FromEmail),
	}

	headers := map[string]string{
		"From":         from.String(),
		"To":           strings.TrimSpace(to),
		"Subject":      "Your BoltPe Financial Planning Report",
		"MIME-Version": "1.0",
		"Content-Type": fmt.Sprintf(
			`multipart/mixed; boundary="%s"`,
			writer.Boundary(),
		),
	}

	headerOrder := []string{
		"From",
		"To",
		"Subject",
		"MIME-Version",
		"Content-Type",
	}

	for _, key := range headerOrder {
		value := headers[key]

		if _, err := fmt.Fprintf(
			&body,
			"%s: %s\r\n",
			key,
			value,
		); err != nil {
			return fmt.Errorf("write email header: %w", err)
		}
	}

	body.WriteString("\r\n")

	textHeader := make(textproto.MIMEHeader)
	textHeader.Set("Content-Type", "text/html; charset=UTF-8")
	textHeader.Set(
		"Content-Transfer-Encoding",
		"quoted-printable",
	)

	htmlPart, err := writer.CreatePart(textHeader)
	if err != nil {
		return fmt.Errorf("create HTML email part: %w", err)
	}

	quotedWriter := quotedprintable.NewWriter(htmlPart)

	safeCustomerName := strings.TrimSpace(customerName)
	if safeCustomerName == "" {
		safeCustomerName = "Customer"
	}

	safeCustomerName = html.EscapeString(safeCustomerName)

	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta
		name="viewport"
		content="width=device-width, initial-scale=1.0"
	>
	<title>BoltPe Financial Planning Report</title>
</head>

<body style="
	margin:0;
	padding:0;
	background-color:#f7eee8;
	font-family:'BoltPe', Arial, Helvetica, sans-serif;
	color:#14293d;
">
	<table
		role="presentation"
		width="100%%"
		cellspacing="0"
		cellpadding="0"
		border="0"
		style="
			width:100%%;
			background-color:#f7eee8;
			font-family:'BoltPe', Arial, Helvetica, sans-serif;
		"
	>
		<tr>
			<td
				align="center"
				style="padding:24px 16px;"
			>
				<table
					role="presentation"
					width="100%%"
					cellspacing="0"
					cellpadding="0"
					border="0"
					style="
						width:100%%;
						max-width:760px;
						background-color:#ffffff;
						border:1px solid #ead5c8;
						border-radius:22px;
						font-family:'BoltPe', Arial, Helvetica, sans-serif;
					"
				>
					<tr>
						<td style="padding:32px 34px 30px 34px;">

							<h1 style="
								margin:0 0 24px 0;
								font-family:'BoltPe', Arial, Helvetica, sans-serif;
								font-size:30px;
								line-height:1.25;
								font-weight:700;
								color:#8F3B1A;
							">
								BoltPe Financial Planning Report
							</h1>

							<p style="
								margin:0 0 18px 0;
								font-family:'BoltPe', Arial, Helvetica, sans-serif;
								font-size:19px;
								line-height:1.45;
								color:#14293d;
							">
								Dear %s,
							</p>

							<p style="
								margin:0 0 18px 0;
								font-family:'BoltPe', Arial, Helvetica, sans-serif;
								font-size:19px;
								line-height:1.45;
								color:#14293d;
							">
								Thank you for using our BoltPe Financial Goal Planner.
							</p>

							<p style="
								margin:0 0 18px 0;
								font-family:'BoltPe', Arial, Helvetica, sans-serif;
								font-size:19px;
								line-height:1.45;
								color:#14293d;
							">
								Your personalized BoltPe Financial Planning Report
								has been generated successfully.
							</p>

							<p style="
								margin:0 0 22px 0;
								font-family:'BoltPe', Arial, Helvetica, sans-serif;
								font-size:19px;
								line-height:1.45;
								color:#14293d;
							">
								Please find the complete PDF report attached with
								this email.
							</p>

							<table
								role="presentation"
								width="100%%"
								cellspacing="0"
								cellpadding="0"
								border="0"
								style="
									width:100%%;
									margin-top:8px;
									background-color:#fff7f2;
									border-left:7px solid #8F3B1A;
									border-radius:14px;
								"
							>
								<tr>
									<td style="padding:16px 18px;">
										<p style="
											margin:0;
											font-family:'BoltPe', Arial, Helvetica, sans-serif;
											font-size:17px;
											line-height:1.45;
											font-weight:600;
											color:#14293d;
										">
											Regards,<br>
											<span style="color:#8F3B1A;">
												BoltPe Financial Planning Team
											</span>
										</p>
									</td>
								</tr>
							</table>

							<p style="
								margin:22px 0 0 0;
								font-family:'BoltPe', Arial, Helvetica, sans-serif;
								font-size:14px;
								line-height:1.45;
								font-weight:600;
								color:#687a99;
							">
								This report is for financial planning assistance
								only. Please review the disclaimer in the PDF before
								making any financial decision.
							</p>

						</td>
					</tr>
				</table>
			</td>
		</tr>
	</table>
</body>
</html>`,
		safeCustomerName,
	)

	if _, err := quotedWriter.Write([]byte(htmlBody)); err != nil {
		return fmt.Errorf("write HTML email body: %w", err)
	}

	if err := quotedWriter.Close(); err != nil {
		return fmt.Errorf("close HTML email writer: %w", err)
	}

	attachmentHeader := make(textproto.MIMEHeader)
	attachmentHeader.Set(
		"Content-Type",
		`application/pdf; name="`+filename+`"`,
	)
	attachmentHeader.Set(
		"Content-Disposition",
		fmt.Sprintf(
			`attachment; filename="%s"`,
			filename,
		),
	)
	attachmentHeader.Set(
		"Content-Transfer-Encoding",
		"base64",
	)

	attachmentPart, err := writer.CreatePart(attachmentHeader)
	if err != nil {
		return fmt.Errorf("create PDF attachment part: %w", err)
	}

	base64Writer := base64.NewEncoder(
		base64.StdEncoding,
		attachmentPart,
	)

	if _, err := base64Writer.Write(pdfData); err != nil {
		return fmt.Errorf("encode PDF attachment: %w", err)
	}

	if err := base64Writer.Close(); err != nil {
		return fmt.Errorf("close PDF encoder: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("close multipart email writer: %w", err)
	}

	address := net.JoinHostPort(
		m.Config.Host,
		fmt.Sprintf("%d", m.Config.Port),
	)

	auth := smtp.PlainAuth(
		"",
		m.Config.Username,
		m.Config.Password,
		m.Config.Host,
	)

	tlsConfig := &tls.Config{
		ServerName: m.Config.Host,
		MinVersion: tls.VersionTLS12,
	}

	if !m.Config.UseTLS && !m.Config.UseSSL {
		if err := smtp.SendMail(
			address,
			auth,
			m.Config.FromEmail,
			[]string{to},
			body.Bytes(),
		); err != nil {
			return fail("send SMTP message", err)
		}

		slog.Info(
			"SMTP server accepted email",
			"from", fromEmail,
			"to", to,
			"duration_ms", time.Since(startedAt).Milliseconds(),
			"delivery_note", "accepted by SMTP server; final inbox delivery depends on the recipient server",
		)
		return nil
	}

	var client *smtp.Client

	if m.Config.UseSSL {
		connection, dialErr := tls.Dial(
			"tcp",
			address,
			tlsConfig,
		)
		if dialErr != nil {
			return fmt.Errorf(
				"connect to SMTP server using SSL: %w",
				dialErr,
			)
		}

		client, err = smtp.NewClient(
			connection,
			m.Config.Host,
		)
		if err != nil {
			_ = connection.Close()

			return fmt.Errorf(
				"create SMTP SSL client: %w",
				err,
			)
		}
	} else {
		client, err = smtp.Dial(address)
		if err != nil {
			return fmt.Errorf(
				"connect to SMTP server: %w",
				err,
			)
		}

		if err := client.StartTLS(tlsConfig); err != nil {
			_ = client.Close()

			return fmt.Errorf(
				"start SMTP TLS: %w",
				err,
			)
		}
	}

	defer func() {
		_ = client.Close()
	}()

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf(
			"SMTP authentication failed: %w",
			err,
		)
	}

	if err := client.Mail(m.Config.FromEmail); err != nil {
		return fmt.Errorf(
			"set sender address: %w",
			err,
		)
	}

	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf(
			"set recipient address: %w",
			err,
		)
	}

	dataWriter, err := client.Data()
	if err != nil {
		return fmt.Errorf(
			"open SMTP message writer: %w",
			err,
		)
	}

	if _, err := dataWriter.Write(body.Bytes()); err != nil {
		_ = dataWriter.Close()

		return fmt.Errorf(
			"write SMTP message: %w",
			err,
		)
	}

	if err := dataWriter.Close(); err != nil {
		return fmt.Errorf(
			"close SMTP message writer: %w",
			err,
		)
	}

	if err := client.Quit(); err != nil {
		return fmt.Errorf(
			"close SMTP connection: %w",
			err,
		)
	}

	slog.Info(
		"SMTP server accepted email",
		"from", fromEmail,
		"to", to,
		"duration_ms", time.Since(startedAt).Milliseconds(),
		"delivery_note", "accepted by SMTP server; final inbox delivery depends on the recipient server",
	)
	return nil
}

func ValidateAddress(value string) error {
	value = strings.TrimSpace(value)

	if value == "" {
		return fmt.Errorf("email address is empty")
	}

	parsedAddress, err := mail.ParseAddress(value)
	if err != nil {
		return fmt.Errorf(
			"invalid email address: %w",
			err,
		)
	}

	if !strings.EqualFold(
		parsedAddress.Address,
		value,
	) {
		return fmt.Errorf(
			"email address must not contain a display name",
		)
	}

	return nil
}

func DialCheck(host string, port int) error {
	host = strings.TrimSpace(host)

	if host == "" {
		return fmt.Errorf("SMTP host is empty")
	}

	if port <= 0 {
		return fmt.Errorf("invalid SMTP port: %d", port)
	}

	connection, err := net.Dial(
		"tcp",
		net.JoinHostPort(
			host,
			fmt.Sprintf("%d", port),
		),
	)
	if err != nil {
		return err
	}

	return connection.Close()
}

// SendTestEmail sends a lightweight SMTP test message without any report attachment.
func (m Mailer) SendTestEmail(to string) error {
	startedAt := time.Now()
	to = strings.TrimSpace(to)
	fromEmail := strings.TrimSpace(m.Config.FromEmail)

	fail := func(stage string, err error) error {
		wrapped := fmt.Errorf("%s: %w", stage, err)
		slog.Error(
			"SMTP test email failed",
			"stage", stage,
			"smtp_host", m.Config.Host,
			"smtp_port", m.Config.Port,
			"from", fromEmail,
			"to", to,
			"duration_ms", time.Since(startedAt).Milliseconds(),
			"error", err,
		)
		return wrapped
	}

	if err := ValidateAddress(to); err != nil {
		return fail("validate recipient email", err)
	}
	if err := ValidateAddress(m.Config.FromEmail); err != nil {
		return fail("validate sender email", err)
	}
	if strings.TrimSpace(m.Config.Host) == "" {
		return fail("validate SMTP configuration", fmt.Errorf("SMTP host is empty"))
	}
	if m.Config.Port <= 0 {
		return fail("validate SMTP configuration", fmt.Errorf("invalid SMTP port: %d", m.Config.Port))
	}
	if strings.TrimSpace(m.Config.Username) == "" {
		return fail("validate SMTP configuration", fmt.Errorf("SMTP username is empty"))
	}
	if strings.TrimSpace(m.Config.Password) == "" {
		return fail("validate SMTP configuration", fmt.Errorf("SMTP password is empty"))
	}

	from := mail.Address{Name: strings.TrimSpace(m.Config.FromName), Address: fromEmail}
	message := strings.Join([]string{
		"From: " + from.String(),
		"To: " + to,
		"Subject: BoltPe Financial Planner - SMTP Test",
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		"<!doctype html><html><body><h2>BoltPe Financial Planner</h2><p>This is a test email from the BoltPe Financial Planner SMTP service.</p><p>If you received this message, SMTP connectivity, authentication, sender acceptance and recipient acceptance were successful.</p></body></html>",
		"",
	}, "\r\n")

	address := net.JoinHostPort(m.Config.Host, fmt.Sprintf("%d", m.Config.Port))
	auth := smtp.PlainAuth("", m.Config.Username, m.Config.Password, m.Config.Host)
	tlsConfig := &tls.Config{ServerName: m.Config.Host, MinVersion: tls.VersionTLS12}

	if !m.Config.UseTLS && !m.Config.UseSSL {
		if err := smtp.SendMail(address, auth, fromEmail, []string{to}, []byte(message)); err != nil {
			return fail("send SMTP test message", err)
		}
		slog.Info("SMTP test email accepted", "from", fromEmail, "to", to, "duration_ms", time.Since(startedAt).Milliseconds())
		return nil
	}

	var client *smtp.Client
	var err error
	if m.Config.UseSSL {
		connection, dialErr := tls.Dial("tcp", address, tlsConfig)
		if dialErr != nil {
			return fail("connect to SMTP server using SSL", dialErr)
		}
		client, err = smtp.NewClient(connection, m.Config.Host)
		if err != nil {
			_ = connection.Close()
			return fail("create SMTP SSL client", err)
		}
	} else {
		client, err = smtp.Dial(address)
		if err != nil {
			return fail("connect to SMTP server", err)
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			_ = client.Close()
			return fail("start SMTP TLS", err)
		}
	}
	defer func() { _ = client.Close() }()

	if err := client.Auth(auth); err != nil {
		return fail("SMTP authentication failed", err)
	}
	if err := client.Mail(fromEmail); err != nil {
		return fail("set sender address", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fail("set recipient address", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fail("open SMTP message writer", err)
	}
	if _, err := writer.Write([]byte(message)); err != nil {
		_ = writer.Close()
		return fail("write SMTP test message", err)
	}
	if err := writer.Close(); err != nil {
		return fail("close SMTP message writer", err)
	}
	if err := client.Quit(); err != nil {
		return fail("close SMTP connection", err)
	}

	slog.Info("SMTP test email accepted", "from", fromEmail, "to", to, "duration_ms", time.Since(startedAt).Milliseconds())
	return nil
}
