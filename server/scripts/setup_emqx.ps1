param([string]$Action = "setup")

$EmqxHost = "192.168.50.194"
$Port = "18083"
$Url = "http://$EmqxHost`:$Port"
$User = "admin"
$Pass = "xuehua123"

function Info($msg) { Write-Host "[INFO] $msg" -ForegroundColor Green }
function Warn($msg) { Write-Host "[WARN] $msg" -ForegroundColor Yellow }
function Error($msg) { Write-Host "[ERROR] $msg" -ForegroundColor Red }

function Test-Connection {
    Info "Testing EMQX connection..."
    try {
        $response = Invoke-WebRequest -Uri $Url -TimeoutSec 10 -UseBasicParsing
        if ($response.StatusCode -eq 200) {
            Info "EMQX Dashboard connection successful"
            return $true
        }
    }
    catch {
        Error "Connection failed: $($_.Exception.Message)"
        return $false
    }
}

function Get-Token {
    Info "Getting API token..."
    try {
        $body = @{ username = $User; password = $Pass } | ConvertTo-Json
        $response = Invoke-RestMethod -Uri "$Url/api/v5/login" -Method Post -ContentType "application/json" -Body $body
        if ($response.token) {
            Info "Token obtained successfully"
            return $response.token
        }
    }
    catch {
        Error "Failed to get token: $($_.Exception.Message)"
        return $null
    }
}

function Set-Config($token) {
    Info "Configuring JWT authentication..."
    
    $config = @{
        mechanism = "password_based"
        backend = "jwt"
        enable = $true
        use_jwks = $false
        algorithm = "hmac-based"
        secret = "78c0f08f-9663-4c9c-a399-cc4ec36b8112"
        secret_base64_encoded = $false
        from = "password"
        verify_claims = @{
            exp = "`${timestamp}"
            iss = "qmPlus"
            aud = "GVA"
            client_id = "`${clientid}"
        }
        disconnect_after_expire = $true
    } | ConvertTo-Json -Depth 10
    
    try {
        $headers = @{ "Authorization" = "Bearer $token"; "Content-Type" = "application/json" }
        Invoke-RestMethod -Uri "$Url/api/v5/authentication" -Method Post -Headers $headers -Body $config | Out-Null
        Info "JWT authentication configured successfully"
    }
    catch {
        Warn "JWT configuration may have failed: $($_.Exception.Message)"
    }
}

function Show-Info {
    Write-Host ""
    Write-Host "=== EMQX Connection Info ===" -ForegroundColor Blue
    Write-Host "Host: $EmqxHost" -ForegroundColor Green
    Write-Host "Dashboard: $Url" -ForegroundColor Green
    Write-Host "MQTT TCP: mqtt://$EmqxHost`:1883" -ForegroundColor Green
    Write-Host "MQTT SSL: mqtts://$EmqxHost`:8883" -ForegroundColor Green
    Write-Host "Username: $User" -ForegroundColor Green
    Write-Host "Password: $Pass" -ForegroundColor Green
    Write-Host ""
    Write-Host "Client Configuration:" -ForegroundColor Yellow
    Write-Host "- Auth Method: JWT (password field)" -ForegroundColor White
    Write-Host "- Client ID: Use clientID from login" -ForegroundColor White
    Write-Host "- Username: clientID" -ForegroundColor White
    Write-Host "- Password: JWT Token" -ForegroundColor White
}

# Main execution
Write-Host "EMQX Remote Setup - NFC Card Relay System" -ForegroundColor Blue
Write-Host "=========================================" -ForegroundColor Blue

switch ($Action.ToLower()) {
    "setup" {
        if (Test-Connection) {
            $token = Get-Token
            if ($token) {
                Set-Config $token
                Show-Info
                Info "EMQX setup completed!"
            }
        }
    }
    "test" {
        if (Test-Connection) {
            $token = Get-Token
            if ($token) {
                Info "Connection and authentication test successful!"
            }
        }
    }
    "info" {
        Show-Info
    }
    "help" {
        Write-Host "Usage: .\setup_emqx.ps1 [command]"
        Write-Host ""
        Write-Host "Commands:"
        Write-Host "  setup   - Configure remote EMQX instance (default)"
        Write-Host "  test    - Test connection and configuration"
        Write-Host "  info    - Show connection information"
        Write-Host "  help    - Show this help"
    }
    default {
        Error "Unknown command: $Action"
        Write-Host "Use '.\setup_emqx.ps1 help' for available commands"
    }
} 