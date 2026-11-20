package slackbot

import "regexp"

const pastedSecretMessage = "🔐 依頼かスレッドにトークンや秘密鍵らしい文字列があるため、Codex には送らず作業を止めました。そのメッセージを削除し、トークンは失効させて作り直してください。作業でトークンが要るときは、ebi-x の運用者に secret として登録してもらいます。"

// Only formats a service issues are matched, never "long random-looking
// strings": a false positive refuses an ordinary request, and hashes, IDs and
// URLs in requests look random too.
var pastedSecretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\bxox[abposr]-[0-9A-Za-z-]{10,}`),
	regexp.MustCompile(`\bxapp-[0-9]-[0-9A-Za-z-]{10,}`),
	regexp.MustCompile(`\bgh[pousr]_[0-9A-Za-z]{36,}`),
	regexp.MustCompile(`\bgithub_pat_[0-9A-Za-z_]{22,}`),
	regexp.MustCompile(`\bglpat-[0-9A-Za-z_-]{20,}`),
	regexp.MustCompile(`\bsk-(?:proj-|ant-)?[0-9A-Za-z_-]{20,}`),
	regexp.MustCompile(`\b[rs]k_live_[0-9A-Za-z]{20,}`),
	regexp.MustCompile(`\b(?:AKIA|ASIA)[0-9A-Z]{16}\b`),
	regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{35}`),
	regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`),
}

func containsPastedSecret(texts ...string) bool {
	for _, text := range texts {
		for _, pattern := range pastedSecretPatterns {
			if pattern.MatchString(text) {
				return true
			}
		}
	}
	return false
}
