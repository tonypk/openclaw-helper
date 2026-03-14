use serde::Serialize;
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
