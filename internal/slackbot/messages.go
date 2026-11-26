package slackbot

import (
	"fmt"

	"github.com/n-seiji/henge/internal/attachment"
)

type messages struct {
	aliveReply                string
	aliveNoCodexReply         string
	aliveCodexDownReply       string
	approvalRequestedMessage  string
	approvalQueuedMessage     string
	notApproverMessage        string
	approvalUsageMessage      string
	approvalSaveFailMessage   string
	claimFailureMessage       string
	threadFailureMessage      string
	workStartFailureMessage   string
	workFailureMessage        string
	forbiddenMessage          string
	attachmentFailureNotice   string
	attachmentOutputNotice    string
	workingStatus             string
	queuedStatus              string
	stoppedMessage            string
	nothingToStopMessage      string
	channelRulesBrokenNotice  string
	doneWatchStartedNotice    string
	doneWatchAllDoneNotice    string
	doneWatchNoTargetNotice   string
	doneWatchLimitNotice      string
	doneWatchTooManyNotice    string
	doneWatchScopeNotice      string
	doneWatchOutsideNotice    string
	doneWatchNamedOutsideNote string
	doneWatchAllOutsideNotice string
	doneWatchFailedNotice     string
	doneWatchReminder         string
	doneWatchLastReminder     string
	doneWatchCompleted        string
	doneWatchGoneMessage      string
	doneWatchStoppedMessage   string
	doneWatchNothingToStop    string
	deferredMessage           string
	deferredStoppedMessage    string
	repeatedFailureAlert      string
	followUpStartMessage      string
	followUpScheduledNotice   string
	followUpCancelledNotice   string
	followUpInvalidNotice     string
	followUpTimeNotice        string
	followUpChainNotice       string
	followUpStoppedMessage    string
	homeTitle                 string
	homeForbidden             string
	homeEmpty                 string
	homeSettingsTitle         string
	homeSettingsHelp          string
	playbookNoteInvalidNotice string
	playbookFoldedNotice      string
	playbookFoldSkipped       string
	playbookFoldFailed        string
	playbookFoldAdminPost     string
	pullWatchStartedNotice    string
	pullWatchDisabledNotice   string
	pullWatchInvalidNotice    string
	pullWatchFailedNotice     string
	pullWatchClosedNotice     string
	pullWatchRepoNotice       string
	pullWatchTakenNotice      string
	pullWatchTooManyNotice    string
	pullWatchWakeMessage      string
	pullMergedMessage         string
	pullClosedMessage         string
	pullChecksPassedMessage   string
	pullWatchExpiredMessage   string
	pullWatchLimitMessage     string
	pullWatchStoppedMessage   string
	reminderSetNotice         string
	reminderStopNotice        string
	reminderInvalidNotice     string
	reminderTimeNotice        string
	reminderLimitNotice       string
	reminderSourceNotice      string
	reminderMessage           string
	reminderLateSuffix        string
	reminderStoppedMsg        string
	scheduleSetNotice         string
	scheduleStopNotice        string
	scheduleInvalidNotice     string
	scheduleSpecNotice        string
	scheduleLimitNotice       string
	scheduleSourceNotice      string
	scheduleStartMessage      string
	scheduleOriginLink        string
	scheduleStoppedMsg        string
	settingsBrokenNotice      string
	shortcutForbidden         string
	usageLowMessage           string
	usageExhaustedMessage     string
	usageResetSuffix          string
	usageSummaryPrefix        string
	usageUnknownReset         string
	usageReplyPrefix          string
	usageUnavailableReply     string
	usageDailyHeader          string
	usageDailyLine            string
	usageHourlyLine           string
	usageHourlyResetAt        string
	usageLimitFailureMessage  string
	usageLimitRetryAt         string
	usageLimitRetryLater      string
	usageLimitAdminAlert      string
	usageResetsAt             string
	usageReplyReset           string
	usageNotes                string
	listSeparator             string
	durationWeek              string
	durationDays              string
	durationHours             string
	durationMinutes           string
	pastedSecret              string
	authFailureMessage        string
	authAdminAlert            string
	adminNotifiedText         string
	adminContactText          string
	userSettingsSavedNotice   string
	userSettingsClearedNotice string
	userSettingsFailedNotice  string
	approveWord               string
	denyWord                  string
	approvalSubjectUser       string
	approvalSubjectChannel    string
	approvalSubjectWorkflow   string
	approvalYou               string
	approvalThisChannel       string
	approvalThisWorkflow      string
	approvalAnd               string
	approvalRequestLink       string
	approvalRequestLinkFor    string
	approvalRequestLinks      string
	approvalWhereChannel      string
	approvalWhereDM           string
	approvalRequestPost       string
	approvalAlreadyDecided    string
	approvalApprovedNotice    string
	approvalMentionAgain      string
	approvalWaitingOthers     string
	approvalStarting          string
	approvalDecided           string
	approvalApprovedBy        string
	approvalDeniedBy          string
	attachmentReasons         map[attachment.Reason]string
	workDone                  string
	memoryUpdated             string
	defaultContact            string
	doneWatchEveryDay         string
	doneWatchEveryOtherDay    string
	doneWatchEveryNDays       string
	doneWatchOriginal         string
	doneWatchSummary          string
	homeCI                    string
	homeRunning               string
	homeRunningSince          string
	homeFollowUps             string
	homeFollowUpAt            string
	homeWatches               string
	homeSchedules             string
	homeReminders             string
	homeDoneWatches           string
	homeUpdated               string
	homeCount                 string
	homeMore                  string
	homeChannelThread         string
	homeDMThread              string
	checksLabels              map[string]string
	playbookNoteFailed        string
	playbookNoteMissing       string
	playbookNoteSaved         string
	statusCommand             string
	statusCommandPlain        string
	statusMoreFiles           string
	statusEditing             string
	statusSearching           string
	statusTool                string
	statusCount               string
	statusSummarizing         string
	statusStep                string
	pullEventCIFailed         string
	pullEventActivity         string
	pullEventSeparator        string
	scheduleSummary           string
	requestPost               string
	requestPlaceholder        string
	requestTarget             string
	requestLabel              string
	requestTitle              string
	requestSubmit             string
	requestClose              string
	waitingPrefix             string
	waitingReply              string
	waitingMention            string
	unknownCause              string
	suggestedPrompts          []suggestedPrompt
}

const defaultBotName = "henge"

// Tests and code that runs before the Slack name is known use these.
var (
	japanese = japaneseCatalog(defaultBotName)
	english  = englishCatalog(defaultBotName)
)

func japaneseCatalog(bot string) messages {
	return messages{
		aliveReply:                "🟢 生きてます。Codex にもつながっています。",
		aliveNoCodexReply:         "🟢 生きてます。",
		aliveCodexDownReply:       "⚠️ 生きてますが、Codex に接続できません。今は作業を受けられないかもしれません。",
		approvalRequestedMessage:  "🔒 %sには " + bot + " を使う権限がまだないため、承認者に許可を依頼しました%s。\n許可されたら、このスレッドでお知らせします。そのあと、もう一度 mention してください。",
		approvalQueuedMessage:     "🔒 %sには " + bot + " を使う権限がまだないため、承認者に許可を依頼しました%s。\nそのままお待ちください！許可されたら、この依頼から自動で始めます。",
		notApproverMessage:        "承認者（SLACK_APPROVER_USER_IDS）だけが操作できます。",
		approvalUsageMessage:      "許可や拒否は、依頼のスレッドで「" + approveCommand + "」か「" + denyCommand + "」とだけ mention してください。",
		approvalSaveFailMessage:   "記録の保存に失敗したため、操作していません。もう一度お試しください。",
		claimFailureMessage:       "受付に失敗しました。お手数ですが、もう一度 mention してください。",
		threadFailureMessage:      "⚠️ スレッドの読み込みに失敗したため、作業は開始していません。もう一度 mention してください。",
		workStartFailureMessage:   "⚠️ 作業を開始できなかったため、作業は行っていません。もう一度 mention してください。",
		workFailureMessage:        "⚠️ 作業が完了したことを確認できませんでした。状況を確認し、新しい mention で依頼し直してください。",
		forbiddenMessage:          "🔒 %sには " + bot + " を使う権限がありません。使いたい場合は %s に許可を頼んでください。",
		attachmentFailureNotice:   "⚠️ 次のファイルを添付できませんでした。生成済みのファイルは作業領域に残しています。再送する場合は「添付を再送して」と依頼してください。",
		attachmentOutputNotice:    "⚠️ 添付ファイルの指定を読み取れなかったため、ファイルは添付していません。再送する場合は「添付を再送して」と依頼してください。",
		workingStatus:             "が作業を進めています…",
		queuedStatus:              "が作業の順番を待っています…",
		stoppedMessage:            "⏹️ 依頼により作業を停止しました。途中までの変更が作業領域に残っている場合があります。続ける場合は、新しい mention で依頼してください。",
		nothingToStopMessage:      "停止する作業はありません。",
		channelRulesBrokenNotice:  "⚠️ チャンネルごとの設定ファイルを読めません（%v）。直すまで、承認チャンネル以外の依頼は始められません。",
		doneWatchStartedNotice:    "👀 :done: の監視を始めます。対象は元メッセージで @メンションされた%d人です。%sに、まだ :done: を付けていない人にこのスレッドで通知します（次回 %s）。%s\n全員が付けるか、%d回通知したら終了します。止める場合は、このスレッドで「@" + bot + " 監視をとめて」と mention してください。",
		doneWatchAllDoneNotice:    "✅ 対象の全員がすでに :done: を付けているため、監視は始めていません。",
		doneWatchNoTargetNotice:   "⚠️ @メンションされた人が見つからなかったため、監視は始めていません。対象の人を @メンションしたメッセージの末尾で「@" + bot + " 監視して」と書いてください。",
		doneWatchLimitNotice:      "⚠️ 1人が同時に続けられる監視は%d件までのため、監視は始めていません。不要な監視を、そのスレッドで「@" + bot + " 監視をとめて」と mention して止めてください。",
		doneWatchTooManyNotice:    "⚠️ 対象が%d人を超えるため、監視は始めていません。",
		doneWatchScopeNotice:      "⚠️ @channel やユーザーグループの対象を調べるには、Slack アプリに channels:read・groups:read・usergroups:read・users:read の scope が必要です。管理者に追加を頼んでください。",
		doneWatchOutsideNotice:    "ユーザーグループのうち、このチャンネルにいない%d人は通知が届かないため対象から外しました。",
		doneWatchNamedOutsideNote: "%s はまだこのチャンネルに参加していないようです。招待しないと通知が届きません。",
		doneWatchAllOutsideNotice: "⚠️ ユーザーグループのメンバーは全員このチャンネルにいないため、通知が届きません。監視は始めていません。メンバーをチャンネルに招待してから、もう一度「@" + bot + " 監視して」と mention してください。",
		doneWatchFailedNotice:     "⚠️ 監視を始められませんでした。もう一度 mention してください。",
		doneWatchReminder:         "%s\n:done: のリアクションがまだのようです。対応が済んだら、%sに :done: を付けてください。（%d回目の通知）",
		doneWatchLastReminder:     "\n%d回通知したため、監視はこれで終了します。",
		doneWatchCompleted:        "✅ 対象の全員が :done: を付けたため、監視を終了しました。",
		doneWatchGoneMessage:      "元メッセージが見つからなくなったため、リアクションの監視を終了しました。",
		doneWatchStoppedMessage:   "リアクションの監視を終了しました。",
		doneWatchNothingToStop:    "このスレッドで続けているリアクションの監視はありません。",
		deferredMessage:           "🔄 " + bot + " を更新しています。いま動いている作業が終わり次第再起動し、この依頼は再起動後に自動で始めます。",
		deferredStoppedMessage:    "再起動後に始める予定だった依頼を取り消しました。",
		repeatedFailureAlert:      "🚨 " + bot + " の作業が直近%d分で%d件失敗しています。依頼者には「依頼し直してください」と返していますが、同じ原因で失敗し続けている可能性があります。" + bot + " のログを確認してください。\n最新のエラー: `%s`",
		followUpStartMessage:      "⏰ 予定していたフォローアップを始めます: %s",
		followUpScheduledNotice:   "⏰ %s ごろにフォローアップします: %s\n止める場合は、このスレッドで「停止」と mention してください。",
		followUpCancelledNotice:   "⏰ 予定していたフォローアップ（%s）は取り消しました。",
		followUpInvalidNotice:     "⚠️ フォローアップの指定を読み取れなかったため、予定していません。",
		followUpTimeNotice:        "⚠️ フォローアップの時刻「%s」を読み取れないか範囲外（%s）のため、予定していません。",
		followUpChainNotice:       "⚠️ 人の発言なしで続けられるフォローアップの上限（%d回）に達したため、次は予定していません。続ける場合は、このスレッドで依頼してください。",
		followUpStoppedMessage:    "予定していたフォローアップを取り消しました。",
		homeTitle:                 "*" + bot + " の作業*\nあなたが依頼した作業の今の状況です。ホームを開き直すと最新になります。",
		homeForbidden:             "*" + bot + " の作業*\n" + bot + " を使う許可がないため、作業はありません。",
		homeEmpty:                 "ありません。",
		homeSettingsTitle:         "⚙️ あなたの個人設定",
		homeSettingsHelp:          bot + " への依頼の中で「返事は短くして、今後もそうして」のように頼むと設定され、「個人設定から〇〇を消して」で変更や削除ができます。この一覧はあなたにだけ表示されます。",
		playbookNoteInvalidNotice: "⚠️ playbook メモの指定を読み取れなかったため、保存していません。",
		playbookFoldedNotice:      "📘 メモが%d件たまったため、playbook「%s」の本文に取り込みました: %s",
		playbookFoldSkipped:       "📘 playbook「%s」のメモ%d件の取り込みを試みましたが、本文は変わりませんでした: %s",
		playbookFoldFailed:        "⚠️ playbook「%s」へのメモの取り込みに失敗しました。メモはそのまま残っています。",
		playbookFoldAdminPost:     "📘 playbook「%s」にメモ%d件を取り込みました（%s）: %s",
		pullWatchStartedNotice:    "👀 次の PR を見守ります。CI の失敗やレビュー・コメントがあれば、このスレッドで対応して報告します。止める場合は、このスレッドで「停止」と mention してください。",
		pullWatchDisabledNotice:   "⚠️ GitHub の見守りが設定されていないため、PR は見守っていません。",
		pullWatchInvalidNotice:    "⚠️ 次の URL は見守れる PR として読み取れなかったため、見守っていません。",
		pullWatchFailedNotice:     "⚠️ 次の PR の状態を GitHub から読めなかったため、見守っていません。",
		pullWatchClosedNotice:     "次の PR はマージ済みか閉じられているため、見守っていません。",
		pullWatchRepoNotice:       "⚠️ 次の PR は " + bot + " が作業するリポジトリのものではないため、見守っていません。",
		pullWatchTakenNotice:      "⚠️ 次の PR は別のスレッドで見守っているため、ここでは見守っていません。",
		pullWatchTooManyNotice:    "⚠️ 一度に見守れる PR は%d件までのため、残りは見守っていません。",
		pullWatchWakeMessage:      "🔔 PR に動きがありました（%s）: %s。対応を始めます。",
		pullMergedMessage:         "🎉 PR がマージされました: %s\nPR の見守りを終了します。",
		pullClosedMessage:         "PR がマージされずに閉じられました: %s\nPR の見守りを終了します。",
		pullChecksPassedMessage:   "✅ PR の CI が通りました: %s",
		pullWatchExpiredMessage:   "⌛ 7日間動きがなかったため、PR の見守りを終了しました: %s",
		pullWatchLimitMessage:     "⚠️ 人の発言なしで PR に対応できる回数の上限（%d回）に達したため、見守りを終了しました: %s\n続ける場合は、このスレッドで依頼してください。",
		pullWatchStoppedMessage:   "PR の見守りを終了しました。",
		reminderSetNotice:         "🔔 %s にこのスレッドでリマインドします: %s\n取り消す場合は、このスレッドで「停止」と mention してください。",
		reminderStopNotice:        "🔔 予定していたリマインダー（%d件）を取り消しました。",
		reminderInvalidNotice:     "⚠️ リマインダーの指定を読み取れなかったため、設定していません。",
		reminderTimeNotice:        "⚠️ リマインダーの時刻「%s」を読み取れないか範囲外（%s）のため、設定していません。",
		reminderLimitNotice:       "⚠️ 1人が予定できるリマインダーは%d件までのため、設定していません。不要なものを、設定したスレッドで「停止」と mention して取り消してください。",
		reminderSourceNotice:      "⚠️ リマインダーは人の依頼への回答でだけ設定・取り消しできるため、指定を無視しました。",
		reminderMessage:           "<@%s> 🔔 リマインダー: %s",
		reminderLateSuffix:        "\n（予定は %s でしたが、bot が止まっていたため遅れました）",
		reminderStoppedMsg:        "予定していたリマインダーを取り消しました。",
		scheduleSetNotice:         "🔁 定期実行を設定しました: %s に「%s」（次回 %s）\n毎回このチャンネルの新しいスレッドで始めます。止める場合は、このスレッドで「停止」と mention してください。",
		scheduleStopNotice:        "🔁 定期実行（%s）を止めました。",
		scheduleInvalidNotice:     "⚠️ 定期実行の指定を読み取れなかったため、設定していません。",
		scheduleSpecNotice:        "⚠️ 定期実行の「%s」を読み取れなかったため、設定していません。「毎日 09:00」「平日 09:00」「週末 10:00」「毎週月水金 09:30」の形で指定できます。",
		scheduleLimitNotice:       "⚠️ 1人が設定できる定期実行は%d件までのため、設定していません。不要な定期実行を、設定したスレッドで「停止」と mention して止めてください。",
		scheduleSourceNotice:      "⚠️ 定期実行は人の依頼への回答でだけ設定・変更できるため、指定を無視しました。",
		scheduleStartMessage:      "🔁 定期実行（%s）: %s",
		scheduleOriginLink:        "\n設定したスレッド: <%s|こちら>（止める場合はそこで「停止」と mention してください）",
		scheduleStoppedMsg:        "定期実行を止めました。",
		settingsBrokenNotice:      "⚠️ " + bot + " の設定ファイルを読めません（%v）。直すまで、既定の設定で動き、SQL の書き込みはすべて止めます。",
		shortcutForbidden:         "このメッセージについて " + bot + " に依頼する許可がありません。",
		usageLowMessage:           "⚠️ Codex の利用枠（%s）の残りが %d%% を下回りました。%s",
		usageExhaustedMessage:     "🚫 Codex の利用枠（%s）を使い切りました。%sまで、" + bot + " は作業できません。",
		usageResetSuffix:          "%sにリセットされます。",
		usageSummaryPrefix:        "今の残り: ",
		usageUnknownReset:         "リセットされる",
		usageReplyPrefix:          "Codex の利用枠の残り: ",
		usageUnavailableReply:     "⚠️ Codex の利用枠を読み取れませんでした。",
		usageDailyHeader:          "📊 %s の Codex の利用枠（%d回確認）",
		usageDailyLine:            "- %s: %d%% → %d%%（最低 %d%%）",
		usageHourlyLine:           "残：%.1f%%",
		usageHourlyResetAt:        "Reset at %s",
		usageLimitFailureMessage:  "⚠️ Codex の利用上限に達したため、作業を完了できませんでした。%s%s、新しい mention で依頼し直してください。途中までの変更が作業領域に残っている場合があります。",
		usageLimitRetryAt:         "%s以降に",
		usageLimitRetryLater:      "利用枠がリセットされてから",
		usageLimitAdminAlert:      "🚫 Codex の利用上限に達したため、" + bot + " は作業できません。利用枠がリセットされるまでお待ちください。",
		usageResetsAt:             "%sにリセットされる",
		usageReplyReset:           "（%sにリセット）",
		usageNotes:                "（%s）",
		listSeparator:             "・",
		durationWeek:              "1週間",
		durationDays:              "%d日",
		durationHours:             "%d時間",
		durationMinutes:           "%d分",
		pastedSecret:              "🔐 依頼かスレッドにトークンや秘密鍵らしい文字列があるため、Codex には送らず作業を止めました。そのメッセージを削除し、トークンは失効させて作り直してください。",
		authFailureMessage:        "⚠️ " + bot + " の Codex のログインが切れているため、作業できませんでした。%s復旧したら、新しい mention で依頼し直してください。",
		authAdminAlert:            "🔑 " + bot + " の Codex のログインが切れているため、作業できません。" + bot + " を動かしている環境で `codex login` を実行して、ログインし直してください。",
		adminNotifiedText:         "管理者に知らせたので、お待ちください。",
		adminContactText:          "管理者に連絡してください。",
		userSettingsSavedNotice:   "⚙️ あなたの個人設定を更新しました（%d件）。内容は " + bot + " のアプリのホームで確認できます。このメッセージはあなたにだけ表示されています。",
		userSettingsClearedNotice: "⚙️ あなたの個人設定をすべて消しました。このメッセージはあなたにだけ表示されています。",
		userSettingsFailedNotice:  "⚠️ 個人設定を保存できませんでした。もう一度お試しください。このメッセージはあなたにだけ表示されています。",
		approveWord:               approveCommand,
		denyWord:                  denyCommand,
		approvalSubjectUser:       "ユーザー <@%s>",
		approvalSubjectChannel:    "チャンネル <#%s>",
		approvalSubjectWorkflow:   "Workflow `%s`",
		approvalYou:               "あなた",
		approvalThisChannel:       "このチャンネル",
		approvalThisWorkflow:      "この Workflow",
		approvalAnd:               "と",
		approvalRequestLink:       "（<%s|承認依頼>）",
		approvalRequestLinkFor:    "<%s|%sの承認依頼>",
		approvalRequestLinks:      "（%s）",
		approvalWhereChannel:      "<#%s> で mention しました。",
		approvalWhereDM:           bot + " に DM で依頼しました。",
		approvalRequestPost:       "🔐 利用許可の依頼\n対象: %s\n<@%s> が %s\n\nこのスレッドで " + bot + " に mention して「%s」か「%s」と返信してください。",
		approvalAlreadyDecided:    "%s は、すでに%s。",
		approvalApprovedNotice:    "✅ %s が許可されました。",
		approvalMentionAgain:      "もう一度 mention してください。",
		approvalWaitingOthers:     "残りの許可を待っています。",
		approvalStarting:          "依頼を始めます。",
		approvalDecided:           "%s %s: %s。",
		approvalApprovedBy:        "<@%s> が許可しました",
		approvalDeniedBy:          "<@%s> が拒否しました",
		attachmentReasons: map[attachment.Reason]string{
			attachment.TooMany:        fmt.Sprintf("1回の添付は%d件までです", attachment.MaxFiles),
			attachment.NotAllowedType: "添付できない種類のファイルです",
			attachment.NotFound:       "ファイルが見つかりません",
			attachment.Unreadable:     "ファイルを確認できません",
			attachment.OutsideArea:    "このスレッドの作業領域外のファイルです",
			attachment.NotRegular:     "通常のファイルではありません",
			attachment.HardLinked:     "ハードリンクされたファイルです",
			attachment.Empty:          "ファイルが空です",
			attachment.TooLarge:       fmt.Sprintf("%dMBを超えています", attachment.MaxSize>>20),
			attachment.UploadFailed:   "Slackへのアップロードに失敗しました",
		},
		workDone:               "作業が完了しました。",
		memoryUpdated:          "📝 全体メモリを更新しました。",
		defaultContact:         bot + " の管理者",
		doneWatchEveryDay:      "平日の毎日%d時",
		doneWatchEveryOtherDay: "平日1日おきの%d時",
		doneWatchEveryNDays:    "平日%d日ごとの%d時",
		doneWatchOriginal:      "元メッセージ",
		doneWatchSummary:       "対象%d人、%s（次回 %s）",
		homeCI:                 "（CI: %s）",
		homeRunning:            "⏳ 作業中",
		homeRunningSince:       "%s から",
		homeFollowUps:          "⏰ 予定中のフォローアップ",
		homeFollowUpAt:         "%s ごろ: %s",
		homeWatches:            "👀 見守り中の PR",
		homeSchedules:          "🔁 定期実行",
		homeReminders:          "🔔 リマインダー",
		homeDoneWatches:        "✅ リアクションの監視",
		homeUpdated:            "_更新: %s_",
		homeCount:              "*%s（%d）*",
		homeMore:               "\nほか%d件",
		homeChannelThread:      "<#%s> のスレッド",
		homeDMThread:           "DM のスレッド",
		checksLabels:           map[string]string{"pending": "実行中", "success": "成功", "failure": "失敗"},
		playbookNoteFailed:     "⚠️ playbook「%s」のメモを保存できませんでした。",
		playbookNoteMissing:    "⚠️ playbook「%s」が見つからないため、メモを保存していません。",
		playbookNoteSaved:      "📝 playbook「%s」のメモを更新しました: %s",
		statusCommand:          "がコマンドを実行しています（%s）…",
		statusCommandPlain:     "がコマンドを実行しています…",
		statusMoreFiles:        " ほか%d件",
		statusEditing:          "がファイルを編集しています（%s）…",
		statusSearching:        "がWebを検索しています（%s）…",
		statusTool:             "がツールを使っています（%s）…",
		statusCount:            "%d/%d 完了",
		statusSummarizing:      "が作業をまとめています… %s",
		statusStep:             "が「%s」を進めています… %s",
		pullEventCIFailed:      "CI が失敗しました（失敗したチェック: %s）",
		pullEventActivity:      "レビュー・コメントが%d件付きました",
		pullEventSeparator:     "、",
		scheduleSummary:        "%s「%s」（次回 %s）",
		requestPost:            "📝 <@%s> からの依頼（このスレッドのメッセージについて）:\n> %s",
		requestPlaceholder:     "例: このエラーの原因を調べて",
		requestTarget:          "対象のメッセージ: %s",
		requestLabel:           "依頼内容",
		requestTitle:           bot + " に頼む",
		requestSubmit:          "依頼する",
		requestClose:           "やめる",
		waitingPrefix:          "回答待ちです。",
		waitingReply:           "このスレッドに返信してください。",
		waitingMention:         "このスレッドで " + bot + " に mention して答えてください。",
		unknownCause:           "不明",
		suggestedPrompts: []suggestedPrompt{
			{title: "できること", message: bot + " にできることと、依頼の仕方を教えて"},
			{title: "リポジトリの説明", message: "作業できるリポジトリの構成と主な機能を説明して"},
			{title: "最近の変更", message: "作業できるリポジトリの最近の変更をまとめて"},
		},
	}
}
func englishCatalog(bot string) messages {
	return messages{
		aliveReply:                "🟢 I'm alive, and connected to Codex.",
		aliveNoCodexReply:         "🟢 I'm alive.",
		aliveCodexDownReply:       "⚠️ I'm alive, but I can't reach Codex. I may not be able to take work right now.",
		approvalRequestedMessage:  "🔒 %s cannot use " + bot + " yet, so I asked an approver for permission%s.\nI'll let you know in this thread once it's approved. Then mention me again.",
		approvalQueuedMessage:     "🔒 %s cannot use " + bot + " yet, so I asked an approver for permission%s.\nPlease wait! Once it's approved, I'll start on this request automatically.",
		notApproverMessage:        "Only approvers (SLACK_APPROVER_USER_IDS) can do this.",
		approvalUsageMessage:      "To allow or deny, mention me in the request's thread with just \"allow\" or \"deny\".",
		approvalSaveFailMessage:   "I couldn't save the record, so nothing was changed. Please try again.",
		claimFailureMessage:       "I couldn't accept this request. Please mention me again.",
		threadFailureMessage:      "⚠️ I couldn't read the thread, so I haven't started. Please mention me again.",
		workStartFailureMessage:   "⚠️ I couldn't start the work, so nothing was done. Please mention me again.",
		workFailureMessage:        "⚠️ I couldn't confirm that the work finished. Please check the state and ask again in a new mention.",
		forbiddenMessage:          "🔒 %s is not allowed to use " + bot + ". To use it, ask %s for permission.",
		attachmentFailureNotice:   "⚠️ I couldn't attach the files below. The generated files are still in the work area. To try again, ask me to \"resend the attachments\".",
		attachmentOutputNotice:    "⚠️ I couldn't read which files to attach, so nothing was attached. To try again, ask me to \"resend the attachments\".",
		workingStatus:             "is working…",
		queuedStatus:              "is waiting for its turn…",
		stoppedMessage:            "⏹️ Stopped as requested. Changes made so far may remain in the work area. To continue, ask in a new mention.",
		nothingToStopMessage:      "There is no work to stop.",
		channelRulesBrokenNotice:  "⚠️ I can't read the per-channel settings file (%v). Until it is fixed, I can't start requests outside the approval channel.",
		doneWatchStartedNotice:    "👀 Watching for :done:. %d people @mentioned in the original message are covered. %s, I'll remind in this thread anyone who hasn't added :done: yet (next: %s). %s\nThe watch ends when everyone has added it or after %d reminders. To stop it, mention \"@" + bot + " stop watching\" in this thread.",
		doneWatchAllDoneNotice:    "✅ Everyone has already added :done:, so I haven't started watching.",
		doneWatchNoTargetNotice:   "⚠️ I found no one @mentioned, so I haven't started watching. Write \"@" + bot + " watch this\" at the end of a message that @mentions the people to watch.",
		doneWatchLimitNotice:      "⚠️ One person can keep up to %d watches at a time, so I haven't started this one. Stop a watch you no longer need by mentioning \"@" + bot + " stop watching\" in its thread.",
		doneWatchTooManyNotice:    "⚠️ More than %d people would be covered, so I haven't started watching.",
		doneWatchScopeNotice:      "⚠️ To look up who @channel or a user group covers, the Slack app needs the channels:read, groups:read, usergroups:read and users:read scopes. Ask an admin to add them.",
		doneWatchOutsideNotice:    "I left out %d user group members who are not in this channel, since reminders would not reach them.",
		doneWatchNamedOutsideNote: "%s doesn't seem to be in this channel yet. Reminders won't reach them unless they are invited.",
		doneWatchAllOutsideNotice: "⚠️ None of the user group members are in this channel, so reminders would not reach them. I haven't started watching. Invite them to the channel, then mention \"@" + bot + " watch this\" again.",
		doneWatchFailedNotice:     "⚠️ I couldn't start watching. Please mention me again.",
		doneWatchReminder:         "%s\nIt looks like you haven't added :done: yet. When you're done, please add :done: to %s. (reminder %d)",
		doneWatchLastReminder:     "\nThat was reminder %d, so the watch ends here.",
		doneWatchCompleted:        "✅ Everyone has added :done:, so the watch has ended.",
		doneWatchGoneMessage:      "The original message is gone, so the reaction watch has ended.",
		doneWatchStoppedMessage:   "Stopped the reaction watch.",
		doneWatchNothingToStop:    "There is no reaction watch running in this thread.",
		deferredMessage:           "🔄 " + bot + " is updating. It restarts once the running work finishes, and starts this request automatically after the restart.",
		deferredStoppedMessage:    "Cancelled the request that was to start after the restart.",
		repeatedFailureAlert:      "🚨 %[2]d " + bot + " jobs failed in the last %[1]d minutes. Requesters were told to ask again, but the same cause may keep failing them. Check the " + bot + " logs.\nLatest error: `%[3]s`",
		followUpStartMessage:      "⏰ Starting the planned follow-up: %s",
		followUpScheduledNotice:   "⏰ I'll follow up around %s: %s\nTo stop it, mention \"stop\" in this thread.",
		followUpCancelledNotice:   "⏰ Cancelled the planned follow-up (%s).",
		followUpInvalidNotice:     "⚠️ I couldn't read the follow-up, so nothing was planned.",
		followUpTimeNotice:        "⚠️ The follow-up time \"%s\" can't be read or is out of range (%s), so nothing was planned.",
		followUpChainNotice:       "⚠️ The limit of follow-ups in a row without anyone writing (%d) was reached, so no more are planned. To continue, ask in this thread.",
		followUpStoppedMessage:    "Cancelled the planned follow-up.",
		homeTitle:                 "*" + bot + " work*\nThe current state of the work you asked for. Reopen Home to refresh.",
		homeForbidden:             "*" + bot + " work*\nYou are not allowed to use " + bot + ", so there is no work.",
		homeEmpty:                 "None.",
		homeSettingsTitle:         "⚙️ Your personal settings",
		homeSettingsHelp:          "Ask " + bot + " in a request, such as \"keep replies short from now on\", to add a setting, and \"remove … from my personal settings\" to change or remove one. Only you can see this list.",
		playbookNoteInvalidNotice: "⚠️ I couldn't read the playbook note, so it wasn't saved.",
		playbookFoldedNotice:      "📘 %d notes had piled up, so I folded them into the playbook \"%s\": %s",
		playbookFoldSkipped:       "📘 I tried to fold %[2]d notes into the playbook \"%[1]s\", but its text did not change: %[3]s",
		playbookFoldFailed:        "⚠️ I couldn't fold the notes into the playbook \"%s\". The notes are kept as they are.",
		playbookFoldAdminPost:     "📘 Folded %[2]d notes into the playbook \"%[1]s\" (%[3]s): %[4]s",
		pullWatchStartedNotice:    "👀 Watching the PRs below. If CI fails or reviews or comments arrive, I'll handle them and report in this thread. To stop, mention \"stop\" in this thread.",
		pullWatchDisabledNotice:   "⚠️ GitHub watching is not set up, so I'm not watching the PRs.",
		pullWatchInvalidNotice:    "⚠️ These URLs aren't PRs I can watch, so I'm not watching them.",
		pullWatchFailedNotice:     "⚠️ I couldn't read the state of these PRs from GitHub, so I'm not watching them.",
		pullWatchClosedNotice:     "These PRs are merged or closed, so I'm not watching them.",
		pullWatchRepoNotice:       "⚠️ These PRs aren't in a repository " + bot + " works on, so I'm not watching them.",
		pullWatchTakenNotice:      "⚠️ These PRs are watched in another thread, so I'm not watching them here.",
		pullWatchTooManyNotice:    "⚠️ I can watch up to %d PRs at once, so I'm not watching the rest.",
		pullWatchWakeMessage:      "🔔 Something happened on the PR (%s): %s. Starting on it.",
		pullMergedMessage:         "🎉 The PR was merged: %s\nStopped watching it.",
		pullClosedMessage:         "The PR was closed without merging: %s\nStopped watching it.",
		pullChecksPassedMessage:   "✅ CI passed on the PR: %s",
		pullWatchExpiredMessage:   "⌛ Nothing happened for 7 days, so I stopped watching the PR: %s",
		pullWatchLimitMessage:     "⚠️ The limit of PR responses without anyone writing (%d) was reached, so I stopped watching: %s\nTo continue, ask in this thread.",
		pullWatchStoppedMessage:   "Stopped watching the PR.",
		reminderSetNotice:         "🔔 I'll remind you in this thread at %s: %s\nTo cancel, mention \"stop\" in this thread.",
		reminderStopNotice:        "🔔 Cancelled %d planned reminders.",
		reminderInvalidNotice:     "⚠️ I couldn't read the reminder, so nothing was set.",
		reminderTimeNotice:        "⚠️ The reminder time \"%s\" can't be read or is out of range (%s), so nothing was set.",
		reminderLimitNotice:       "⚠️ One person can plan up to %d reminders, so nothing was set. Cancel ones you no longer need by mentioning \"stop\" in the thread where they were set.",
		reminderSourceNotice:      "⚠️ Reminders can only be set or cancelled in an answer to a person's request, so this one was ignored.",
		reminderMessage:           "<@%s> 🔔 Reminder: %s",
		reminderLateSuffix:        "\n(It was due at %s, but the bot was down, so it's late.)",
		reminderStoppedMsg:        "Cancelled the planned reminders.",
		scheduleSetNotice:         "🔁 Scheduled: \"%[2]s\" %[1]s (next: %[3]s)\nEach run starts in a new thread in this channel. To stop it, mention \"stop\" in this thread.",
		scheduleStopNotice:        "🔁 Stopped the schedule (%s).",
		scheduleInvalidNotice:     "⚠️ I couldn't read the schedule, so nothing was set.",
		scheduleSpecNotice:        "⚠️ I couldn't read the schedule \"%s\", so nothing was set. Use a form like \"daily 09:00\", \"weekdays 09:00\", \"weekends 10:00\" or \"Mon,Wed,Fri 09:30\".",
		scheduleLimitNotice:       "⚠️ One person can set up to %d schedules, so nothing was set. Stop ones you no longer need by mentioning \"stop\" in the thread where they were set.",
		scheduleSourceNotice:      "⚠️ Schedules can only be set or changed in an answer to a person's request, so this one was ignored.",
		scheduleStartMessage:      "🔁 Scheduled run (%s): %s",
		scheduleOriginLink:        "\nSet in: <%s|this thread> (mention \"stop\" there to stop it)",
		scheduleStoppedMsg:        "Stopped the schedule.",
		settingsBrokenNotice:      "⚠️ I can't read the " + bot + " settings file (%v). Until it is fixed, I run with the defaults and stop all SQL writes.",
		shortcutForbidden:         "You are not allowed to ask " + bot + " about this message.",
		usageLowMessage:           "⚠️ Less than %[2]d%% of the Codex usage limit (%[1]s) is left. %[3]s",
		usageExhaustedMessage:     "🚫 The Codex usage limit (%s) is used up. " + bot + " can't work until %s.",
		usageResetSuffix:          "It resets at %s.",
		usageSummaryPrefix:        "Left now: ",
		usageUnknownReset:         "it resets",
		usageReplyPrefix:          "Codex usage left: ",
		usageUnavailableReply:     "⚠️ I couldn't read the Codex usage limits.",
		usageDailyHeader:          "📊 Codex usage on %s (checked %d times)",
		usageDailyLine:            "- %s: %d%% → %d%% (lowest %d%%)",
		usageHourlyLine:           "Left: %.1f%%",
		usageHourlyResetAt:        "Reset at %s",
		usageLimitFailureMessage:  "⚠️ The Codex usage limit was reached, so the work couldn't finish. %s%s, ask again in a new mention. Changes made so far may remain in the work area.",
		usageLimitRetryAt:         "After %s",
		usageLimitRetryLater:      "Once the limit resets",
		usageLimitAdminAlert:      "🚫 The Codex usage limit was reached, so " + bot + " can't work. Please wait until the limit resets.",
		usageResetsAt:             "it resets at %s",
		usageReplyReset:           " (resets at %s)",
		usageNotes:                " (%s)",
		listSeparator:             ", ",
		durationWeek:              "1 week",
		durationDays:              "%d days",
		durationHours:             "%d hours",
		durationMinutes:           "%d minutes",
		pastedSecret:              "🔐 The request or thread contains what looks like a token or private key, so I stopped without sending it to Codex. Delete that message, then revoke the token and create a new one.",
		authFailureMessage:        "⚠️ " + bot + "'s Codex login has expired, so the work couldn't be done. %sOnce it is fixed, ask again in a new mention.",
		authAdminAlert:            "🔑 " + bot + "'s Codex login has expired, so it can't work. Run `codex login` where " + bot + " runs to log in again.",
		adminNotifiedText:         "I've told the admin, so please wait. ",
		adminContactText:          "Please contact the admin. ",
		userSettingsSavedNotice:   "⚙️ Updated your personal settings (%d). You can see them on the " + bot + " app's Home. Only you can see this message.",
		userSettingsClearedNotice: "⚙️ Cleared all your personal settings. Only you can see this message.",
		userSettingsFailedNotice:  "⚠️ I couldn't save your personal settings. Please try again. Only you can see this message.",
		approveWord:               "allow",
		denyWord:                  "deny",
		approvalSubjectUser:       "user <@%s>",
		approvalSubjectChannel:    "channel <#%s>",
		approvalSubjectWorkflow:   "Workflow `%s`",
		approvalYou:               "you",
		approvalThisChannel:       "this channel",
		approvalThisWorkflow:      "this Workflow",
		approvalAnd:               " and ",
		approvalRequestLink:       " (<%s|approval request>)",
		approvalRequestLinkFor:    "<%s|approval request for %s>",
		approvalRequestLinks:      " (%s)",
		approvalWhereChannel:      "mentioned " + bot + " in <#%s>.",
		approvalWhereDM:           "asked " + bot + " in a DM.",
		approvalRequestPost:       "🔐 Request for access\nFor: %s\n<@%s> %s\n\nMention " + bot + " in this thread and reply \"%s\" or \"%s\".",
		approvalAlreadyDecided:    "%s was already %s.",
		approvalApprovedNotice:    "✅ %s was allowed.",
		approvalMentionAgain:      "Please mention me again.",
		approvalWaitingOthers:     "Waiting for the remaining approvals.",
		approvalStarting:          "Starting the request.",
		approvalDecided:           "%s %s: %s.",
		approvalApprovedBy:        "allowed by <@%s>",
		approvalDeniedBy:          "denied by <@%s>",
		attachmentReasons: map[attachment.Reason]string{
			attachment.TooMany:        fmt.Sprintf("up to %d files per attachment", attachment.MaxFiles),
			attachment.NotAllowedType: "this kind of file can't be attached",
			attachment.NotFound:       "file not found",
			attachment.Unreadable:     "can't check the file",
			attachment.OutsideArea:    "the file is outside this thread's work area",
			attachment.NotRegular:     "not a regular file",
			attachment.HardLinked:     "the file is hard-linked",
			attachment.Empty:          "the file is empty",
			attachment.TooLarge:       fmt.Sprintf("larger than %dMB", attachment.MaxSize>>20),
			attachment.UploadFailed:   "upload to Slack failed",
		},
		workDone:               "The work is done.",
		memoryUpdated:          "📝 Updated the global memory.",
		defaultContact:         "the " + bot + " admin",
		doneWatchEveryDay:      "Every weekday at %d:00",
		doneWatchEveryOtherDay: "Every other weekday at %d:00",
		doneWatchEveryNDays:    "Every %d weekdays at %d:00",
		doneWatchOriginal:      "the original message",
		doneWatchSummary:       "%d people, %s (next: %s)",
		homeCI:                 " (CI: %s)",
		homeRunning:            "⏳ Running",
		homeRunningSince:       "since %s",
		homeFollowUps:          "⏰ Planned follow-ups",
		homeFollowUpAt:         "around %s: %s",
		homeWatches:            "👀 Watched PRs",
		homeSchedules:          "🔁 Schedules",
		homeReminders:          "🔔 Reminders",
		homeDoneWatches:        "✅ Reaction watches",
		homeUpdated:            "_Updated: %s_",
		homeCount:              "*%s (%d)*",
		homeMore:               "\nand %d more",
		homeChannelThread:      "a thread in <#%s>",
		homeDMThread:           "a DM thread",
		checksLabels:           map[string]string{"pending": "running", "success": "passed", "failure": "failed"},
		playbookNoteFailed:     "⚠️ I couldn't save the note for the playbook \"%s\".",
		playbookNoteMissing:    "⚠️ The playbook \"%s\" wasn't found, so the note wasn't saved.",
		playbookNoteSaved:      "📝 Updated the notes for the playbook \"%s\": %s",
		statusCommand:          "is running a command (%s)…",
		statusCommandPlain:     "is running a command…",
		statusMoreFiles:        " and %d more",
		statusEditing:          "is editing files (%s)…",
		statusSearching:        "is searching the web (%s)…",
		statusTool:             "is using a tool (%s)…",
		statusCount:            "%d/%d done",
		statusSummarizing:      "is wrapping up… %s",
		statusStep:             "is working on \"%s\"… %s",
		pullEventCIFailed:      "CI failed (failed checks: %s)",
		pullEventActivity:      "%d reviews or comments arrived",
		pullEventSeparator:     ", ",
		scheduleSummary:        "%s \"%s\" (next: %s)",
		requestPost:            "📝 Request from <@%s> (about a message in this thread):\n> %s",
		requestPlaceholder:     "e.g. Find the cause of this error",
		requestTarget:          "Message: %s",
		requestLabel:           "Request",
		requestTitle:           "Ask " + bot,
		requestSubmit:          "Ask",
		requestClose:           "Cancel",
		waitingPrefix:          "Waiting for your answer. ",
		waitingReply:           "Please reply in this thread.",
		waitingMention:         "Please answer by mentioning " + bot + " in this thread.",
		unknownCause:           "unknown",
		suggestedPrompts: []suggestedPrompt{
			{title: "What you can do", message: "Tell me what " + bot + " can do and how to ask"},
			{title: "About the repositories", message: "Explain the structure and main features of the repositories you can work on"},
			{title: "Recent changes", message: "Summarize recent changes in the repositories you can work on"},
		},
	}
}
