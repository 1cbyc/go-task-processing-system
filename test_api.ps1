# Go Task Processing System - API Test Script

Write-Host "Testing Go Task Processing System API..." -ForegroundColor Green
Write-Host "==========================================" -ForegroundColor Green

# Test 1: Health Check
Write-Host "`n1. Testing Health Check..." -ForegroundColor Yellow
try {
    $health = Invoke-WebRequest -Uri "http://localhost:8080/health" -Method GET
    Write-Host "Health Status: $($health.Content)" -ForegroundColor Green
} catch {
    Write-Host "Health check failed: $($_.Exception.Message)" -ForegroundColor Red
}

# Test 2: Get Statistics
Write-Host "`n2. Testing Statistics..." -ForegroundColor Yellow
try {
    $stats = Invoke-WebRequest -Uri "http://localhost:8080/stats" -Method GET
    Write-Host "Statistics: $($stats.Content)" -ForegroundColor Green
} catch {
    Write-Host "Statistics failed: $($_.Exception.Message)" -ForegroundColor Red
}

# Test 3: Get Workers
Write-Host "`n3. Testing Workers..." -ForegroundColor Yellow
try {
    $workers = Invoke-WebRequest -Uri "http://localhost:8080/workers" -Method GET
    Write-Host "Workers: $($workers.Content)" -ForegroundColor Green
} catch {
    Write-Host "Workers failed: $($_.Exception.Message)" -ForegroundColor Red
}

# Test 4: Submit Multiple Jobs
Write-Host "`n4. Testing Job Submission..." -ForegroundColor Yellow
$jobTypes = @("email", "report", "backup", "sync", "cleanup")
$priorities = @("low", "normal", "high", "urgent")

for ($i = 0; $i -lt 5; $i++) {
    $jobType = $jobTypes[$i % $jobTypes.Length]
    $priority = $priorities[$i % $priorities.Length]
    
    try {
        $job = Invoke-WebRequest -Uri "http://localhost:8080/jobs?type=$jobType&priority=$priority" -Method POST
        Write-Host "Submitted job: $($job.Content)" -ForegroundColor Green
        Start-Sleep -Milliseconds 200
    } catch {
        Write-Host "Job submission failed: $($_.Exception.Message)" -ForegroundColor Red
    }
}

# Test 5: Final Statistics
Write-Host "`n5. Final Statistics..." -ForegroundColor Yellow
Start-Sleep -Seconds 2
try {
    $finalStats = Invoke-WebRequest -Uri "http://localhost:8080/stats" -Method GET
    Write-Host "Final Statistics: $($finalStats.Content)" -ForegroundColor Green
} catch {
    Write-Host "Final statistics failed: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host "`nAPI Testing Complete!" -ForegroundColor Green 