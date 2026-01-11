# Test API Endpoints

$baseUrl = "http://localhost:8080"

Write-Host "=== Testing Health ===" -ForegroundColor Green
Invoke-RestMethod -Uri "$baseUrl/health" -Method GET

Write-Host "`n=== Testing Subscribe ===" -ForegroundColor Green
$subscribeBody = @{
    user_id = 1
    plan = "monthly"
    duration_months = 1
} | ConvertTo-Json

$headers = @{
    "Content-Type" = "application/json"
    "Idempotency-Key" = "sub-test-001"
}

try {
    Invoke-RestMethod -Uri "$baseUrl/subscribe" -Method POST -Headers $headers -Body $subscribeBody
} catch {
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    $_.ErrorDetails.Message
}

Write-Host "`n=== Testing Get Subscriptions ===" -ForegroundColor Green
Invoke-RestMethod -Uri "$baseUrl/subscriptions/1" -Method GET

Write-Host "`n=== Testing Renew ===" -ForegroundColor Green
$renewBody = @{
    subscription_id = 1
    duration_months = 1
} | ConvertTo-Json

$headers["Idempotency-Key"] = "renew-test-001"

try {
    Invoke-RestMethod -Uri "$baseUrl/renew" -Method POST -Headers $headers -Body $renewBody
} catch {
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    $_.ErrorDetails.Message
}

Write-Host "`n=== Testing Gift Creation ===" -ForegroundColor Green
$giftBody = @{
    gifter_id = 1
    recipient_email = "friend@example.com"
    duration_months = 3
} | ConvertTo-Json

$headers["Idempotency-Key"] = "gift-test-001"

try {
    Invoke-RestMethod -Uri "$baseUrl/gift" -Method POST -Headers $headers -Body $giftBody
} catch {
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    $_.ErrorDetails.Message
}

Write-Host "`n=== Testing Gift Redeem ===" -ForegroundColor Green
$redeemBody = @{
    gift_id = 1
    user_id = 2
} | ConvertTo-Json

$headers["Idempotency-Key"] = "redeem-test-001"

try {
    Invoke-RestMethod -Uri "$baseUrl/gift/redeem" -Method POST -Headers $headers -Body $redeemBody
} catch {
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    $_.ErrorDetails.Message
}

Write-Host "`n=== Testing Cancel ===" -ForegroundColor Green
$cancelBody = @{
    subscription_id = 1
} | ConvertTo-Json

$headers["Idempotency-Key"] = "cancel-test-001"

try {
    Invoke-RestMethod -Uri "$baseUrl/cancel" -Method POST -Headers $headers -Body $cancelBody
} catch {
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    $_.ErrorDetails.Message
}

Write-Host "`n=== All Tests Complete ===" -ForegroundColor Green