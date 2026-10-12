param(
  [Parameter(Mandatory=$true)][string]$ReportName
)
$ErrorActionPreference = 'Stop'
$repo = 'D:\base\TVSOS'
$target = Join-Path $repo $ReportName
$backup = Join-Path $repo 'plans/_report_assets/_word_backup.docx'
$merged = Join-Path $repo 'plans/_report_assets/_merged_report.docx'
if (-not (Test-Path -LiteralPath $target)) { throw 'target docx not found' }
Copy-Item -LiteralPath $target -Destination $backup -Force
Set-Location $repo
python 'plans/_report_assets/_surgery.py'
if ($LASTEXITCODE -ne 0) { throw 'surgery failed' }
if (-not (Test-Path -LiteralPath $merged)) { throw 'merged output missing' }
Move-Item -LiteralPath $merged -Destination $target -Force
Write-Output 'FINALIZED OK'