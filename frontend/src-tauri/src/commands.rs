use serde::{Deserialize, Serialize};
use serde_json::Value;
use tauri_plugin_updater::UpdaterExt;

use crate::helper_ipc;

/// Tauri command that forwards JSON-RPC calls to the Go helper process.
#[tauri::command]
pub async fn helper_rpc(method: String, params: Option<Value>) -> Result<Value, String> {
    helper_ipc::call(&method, params)
        .await
        .map_err(|e| e.to_string())
}

#[derive(Serialize)]
pub struct UpdateInfo {
    pub available: bool,
    pub version: Option<String>,
    pub body: Option<String>,
}

/// Check for application updates.
#[tauri::command]
pub async fn check_update(app: tauri::AppHandle) -> Result<UpdateInfo, String> {
    let updater = app.updater().map_err(|e| e.to_string())?;

    match updater.check().await {
        Ok(Some(update)) => Ok(UpdateInfo {
            available: true,
            version: Some(update.version.clone()),
            body: update.body.clone(),
        }),
        Ok(None) => Ok(UpdateInfo {
            available: false,
            version: None,
            body: None,
        }),
        Err(e) => Err(e.to_string()),
    }
}

/// Telegram fallback — send report directly when Go sidecar is unreachable.
/// Bot token and chat ID are embedded at build time via environment variables.
#[tauri::command]
pub async fn report_telegram_fallback(title: String, body: String) -> Result<bool, String> {
    let bot_token = option_env!("TELEGRAM_BOT_TOKEN").unwrap_or("");
    let chat_id = option_env!("TELEGRAM_CHAT_ID").unwrap_or("");

    if bot_token.is_empty() || chat_id.is_empty() {
        return Err("Telegram not configured in this build".to_string());
    }

    // Escape MarkdownV2 special characters
    let escaped_title = escape_markdownv2(&title);
    let escaped_body = escape_markdownv2(&body);
    let text = format!(
        "🐛 *Report \\(fallback\\)*\n\n*Title:* {}\n\n```\n{}\n```",
        escaped_title, escaped_body
    );

    #[derive(Serialize)]
    struct TelegramMessage {
        chat_id: String,
        text: String,
        parse_mode: String,
    }

    let url = format!("https://api.telegram.org/bot{}/sendMessage", bot_token);
    let client = reqwest::Client::builder()
        .timeout(std::time::Duration::from_secs(10))
        .build()
        .map_err(|e| e.to_string())?;

    let resp = client
        .post(&url)
        .json(&TelegramMessage {
            chat_id: chat_id.to_string(),
            text,
            parse_mode: "MarkdownV2".to_string(),
        })
        .send()
        .await
        .map_err(|e| format!("Telegram request failed: {}", e))?;

    #[derive(Deserialize)]
    struct TelegramResponse {
        ok: bool,
        description: Option<String>,
    }

    let result: TelegramResponse = resp.json().await.map_err(|e| e.to_string())?;
    if !result.ok {
        return Err(format!(
            "Telegram API error: {}",
            result.description.unwrap_or_default()
        ));
    }

    Ok(true)
}

/// Escape special characters for Telegram MarkdownV2.
fn escape_markdownv2(text: &str) -> String {
    let specials = ['_', '*', '[', ']', '(', ')', '~', '`', '>', '#', '+', '-', '=', '|', '{', '}', '.', '!'];
    let mut result = String::with_capacity(text.len() * 2);
    for ch in text.chars() {
        if specials.contains(&ch) {
            result.push('\\');
        }
        result.push(ch);
    }
    result
}
