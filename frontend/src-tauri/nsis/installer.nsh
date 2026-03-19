; Kill running OpenClaw Helper processes before installing/upgrading.
; This prevents "file in use" errors when overwriting och-helper.exe.

!macro NSIS_HOOK_PREINSTALL
  ; First kill the main Tauri app (which spawns och-helper as a sidecar)
  nsExec::ExecToLog 'taskkill /F /IM "OpenClaw Helper.exe"'
  ; Then kill the sidecar itself
  nsExec::ExecToLog 'taskkill /F /IM "och-helper.exe"'

  ; Wait and retry — processes may take a moment to release file handles
  Sleep 2000

  ; Second round kill in case the app respawned the sidecar
  nsExec::ExecToLog 'taskkill /F /IM "och-helper.exe"'
  nsExec::ExecToLog 'taskkill /F /IM "OpenClaw Helper.exe"'

  Sleep 1000

  ; Final check: try to delete the file to confirm it's not locked.
  ; If delete fails, wait longer. ClearErrors resets the error flag.
  ClearErrors
  IfFileExists "$INSTDIR\och-helper.exe" 0 +4
    Delete "$INSTDIR\och-helper.exe"
    IfErrors 0 +2
      Sleep 3000
!macroend
