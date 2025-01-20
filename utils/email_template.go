// utils/email_template.go
package utils

import (
	"bi-backend/config"
	"bytes"
	"html/template"
)

// 验证邮件模板
const verificationEmailTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>验证您的邮箱</title>
</head>
<body style="font-family: 'Microsoft YaHei', Arial, sans-serif;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #333;">欢迎注册BI平台</h2>
        <p>您好！</p>
        <p>请点击下面的按钮验证您的邮箱地址：</p>
        <p style="text-align: center;">
            <a href="{{.VerifyURL}}" style="display: inline-block; padding: 10px 20px; background-color: #4A90E2; color: white; text-decoration: none; border-radius: 5px;">验证邮箱</a>
        </p>
        <p>或者点击以下链接：</p>
        <p><a href="{{.VerifyURL}}">{{.VerifyURL}}</a></p>
        <p>如果这不是您的操作，请忽略此邮件。</p>
        <hr style="border: 1px solid #eee; margin: 20px 0;">
        <p style="color: #666; font-size: 12px;">此邮件由系统自动发送，请勿直接回复。</p>
    </div>
</body>
</html>
`

const resetPasswordEmailTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>重置密码</title>
</head>
<body style="font-family: 'Microsoft YaHei', Arial, sans-serif;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #333;">重置密码请求</h2>
        <p>您好！</p>
        <p>您正在请求重置密码，请点击下面的按钮进行重置：</p>
        <p style="text-align: center;">
            <a href="{{.ResetURL}}" style="display: inline-block; padding: 10px 20px; background-color: #4A90E2; color: white; text-decoration: none; border-radius: 5px;">重置密码</a>
        </p>
        <p>或者点击以下链接：</p>
        <p><a href="{{.ResetURL}}">{{.ResetURL}}</a></p>
        <p>如果这不是您的操作，请忽略此邮件。</p>
        <hr style="border: 1px solid #eee; margin: 20px 0;">
        <p style="color: #666; font-size: 12px;">此邮件由系统自动发送，请勿直接回复。</p>
        <p style="color: #666; font-size: 12px;">出于安全考虑，此链接将在1小时后失效。</p>
    </div>
</body>
</html>
`

func SendVerificationEmail(email, token string) error {
	t, err := template.New("verify").Parse(verificationEmailTemplate)
	if err != nil {
		return err
	}

	var body bytes.Buffer
	err = t.Execute(&body, struct {
		VerifyURL string
	}{
		VerifyURL: config.GlobalConfig.Frontend.URL + "/verify-email?token=" + token,
	})
	if err != nil {
		return err
	}

	return SendEmail(email, "验证您的邮箱", body.String())
}

func SendPasswordResetEmail(email, token string) error {
	t, err := template.New("reset").Parse(resetPasswordEmailTemplate)
	if err != nil {
		return err
	}

	var body bytes.Buffer
	err = t.Execute(&body, struct {
		ResetURL string
	}{
		ResetURL: config.GlobalConfig.Frontend.URL + "/reset-password?token=" + token,
	})
	if err != nil {
		return err
	}

	return SendEmail(email, "重置密码", body.String())
}
