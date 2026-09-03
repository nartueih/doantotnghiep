# Windows VPS Deployment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deploy License Manager at `https://tranhieu1834.duckdns.org` on a Windows Server 2016 VPS with IIS, a Go executable, and a fresh PostgreSQL database.

**Architecture:** IIS terminates HTTP/HTTPS, serves the React build, and proxies `/api` plus `/health` to `127.0.0.1:8080`. The Go API and PostgreSQL 15 run only on the VPS; only TCP 80 and 443 are forwarded from the router.

**Tech Stack:** Windows Server 2016 Desktop Experience, IIS, URL Rewrite, ARR, win-acme, PostgreSQL 15 x64, Go 1.25 build output, React/Vite static build, WinSW.

**Spec:** `docs/superpowers/specs/2026-09-03-windows-vps-deployment-design.md`

## Global Constraints

- Use the fresh VPS database `license_manager`; do not copy the development database.
- Build source code on `D:\DoAn`; the VPS does not need Git, Go, Node.js, or npm.
- Deploy application files under `C:\Deploy\LicenseManager`.
- Bind the API to `127.0.0.1:8080` and PostgreSQL to localhost.
- Forward and allow only TCP 80 and 443 from the Internet.
- Never commit or paste `DATABASE_URL`, `JWT_SECRET`, `LICENSE_ENCRYPTION_KEY`, database passwords, or admin passwords.
- Back up `LICENSE_ENCRYPTION_KEY`; changing or losing it makes stored activation keys unreadable.

---

### Task 1: Capture the VPS baseline and secure network boundary

**Files:**
- Create on VPS: `C:\Deploy\LicenseManager\backups\baseline.txt`

**Interfaces:**
- Consumes: DuckDNS name `tranhieu1834.duckdns.org` and the router's port-forwarding controls.
- Produces: a known VPS LAN address and verified public routing for TCP 80/443.

- [ ] **Step 1: Record the Windows and network baseline**

Run in an elevated Windows PowerShell window on the VPS:

```powershell
New-Item -ItemType Directory -Force -Path 'C:\Deploy\LicenseManager\backups' | Out-Null
$baseline = 'C:\Deploy\LicenseManager\backups\baseline.txt'
Get-ComputerInfo | Select-Object WindowsProductName, WindowsVersion, OsArchitecture | Out-File $baseline
Get-NetIPAddress -AddressFamily IPv4 | Where-Object IPAddress -NotLike '169.254*' | Format-Table InterfaceAlias,IPAddress,PrefixLength | Out-File $baseline -Append
Get-NetIPConfiguration | Format-List InterfaceAlias,IPv4DefaultGateway,DNSServer | Out-File $baseline -Append
Get-Content $baseline
```

Expected: Windows Server 2016 x64 and one stable LAN IPv4 address used by the router.

- [ ] **Step 2: Configure router forwarding**

In the router UI, create exactly these TCP forwards to the VPS LAN IPv4 address recorded in Step 1:

```text
WAN TCP 80  -> VPS TCP 80
WAN TCP 443 -> VPS TCP 443
```

Remove public forwards for TCP 5432, 8080, and 8082 if they exist.

- [ ] **Step 3: Configure Windows Firewall**

Run on the VPS:

```powershell
New-NetFirewallRule -DisplayName 'License Manager HTTP' -Direction Inbound -Action Allow -Protocol TCP -LocalPort 80
New-NetFirewallRule -DisplayName 'License Manager HTTPS' -Direction Inbound -Action Allow -Protocol TCP -LocalPort 443
Get-NetFirewallRule -DisplayName 'License Manager HTTP','License Manager HTTPS' | Select-Object DisplayName,Enabled,Direction,Action
```

Expected: both rules are enabled, inbound, and allowed.

- [ ] **Step 4: Verify DuckDNS from the development machine**

Run on `D:\DoAn`:

```powershell
Resolve-DnsName tranhieu1834.duckdns.org -Type A
```

Expected: `42.114.160.132`, or the current public IP assigned to the router if it has changed.

### Task 2: Install PostgreSQL 15 and create the fresh database

**Files:**
- Managed by installer: PostgreSQL 15 program and data directories.
- Create on VPS: `C:\Deploy\LicenseManager\backups\database-connection-check.txt`

**Interfaces:**
- Consumes: the local PostgreSQL superuser created by the installer.
- Produces: role `license_admin`, database `license_manager`, and local port 5432.

- [ ] **Step 1: Install PostgreSQL 15 x64**

Download PostgreSQL 15 x64 from the official Windows installer page, run the graphical installer as Administrator, and install PostgreSQL Server plus Command Line Tools. Keep port `5432`, locale `Default locale`, and choose a strong password for the installer-created `postgres` account.

- [ ] **Step 2: Confirm the service is running**

Run on the VPS:

```powershell
Get-Service postgresql* | Select-Object Name,Status,StartType
```

Expected: one PostgreSQL 15 service with `Status` equal to `Running` and `StartType` equal to `Automatic`.

- [ ] **Step 3: Create the application role and database**

Run in PowerShell on the VPS. This creates a random 48-character hexadecimal password, displays it once so it can be stored in a password manager, and sends the SQL through psql without placing the password literal in PowerShell history:

```powershell
$databasePasswordBytes = New-Object byte[] 24
$databasePasswordRng = [Security.Cryptography.RandomNumberGenerator]::Create()
$databasePasswordRng.GetBytes($databasePasswordBytes)
$databasePasswordRng.Dispose()
$databasePassword = [BitConverter]::ToString($databasePasswordBytes).Replace('-','').ToLowerInvariant()
Write-Host 'Store this license_admin password now:' $databasePassword
$escapedSqlPassword = $databasePassword.Replace("'","''")
$psql = 'C:\Program Files\PostgreSQL\15\bin\psql.exe'
& $psql -U postgres -d postgres -c "CREATE ROLE license_admin WITH LOGIN PASSWORD '$escapedSqlPassword';"
& $psql -U postgres -d postgres -c 'CREATE DATABASE license_manager OWNER license_admin;'
& $psql -U postgres -d postgres -c 'REVOKE ALL ON DATABASE license_manager FROM PUBLIC;'
$escapedSqlPassword = $null
```

Enter the PostgreSQL `postgres` password chosen during installation when psql prompts for it. Do not paste the generated application password into chat or commit it to Git.

- [ ] **Step 4: Verify the new login locally**

Run on the VPS, entering the application password when prompted:

```powershell
& 'C:\Program Files\PostgreSQL\15\bin\psql.exe' -h 127.0.0.1 -U license_admin -d license_manager -c 'SELECT current_database(), current_user;' | Tee-Object 'C:\Deploy\LicenseManager\backups\database-connection-check.txt'
```

Expected: database `license_manager` and user `license_admin`.

### Task 3: Add and verify IIS deployment artifacts in the repository

**Files:**
- Create: `web/public/web.config`
- Create: `deploy/windows/license-api-service.xml`
- Test: `web/dist/web.config`

**Interfaces:**
- Consumes: backend routes `/api/*` and `/health/*` on port 8080.
- Produces: IIS reverse-proxy/SPA rules and a WinSW service definition named `LicenseManagerApi`.

- [ ] **Step 1: Add the IIS web configuration**

Create `web/public/web.config` with this content:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<configuration>
  <system.webServer>
    <rewrite>
      <rules>
        <rule name="License Manager Backend" stopProcessing="true">
          <match url="^(api|health)(/.*)?$" />
          <action type="Rewrite" url="http://127.0.0.1:8080/{R:0}" />
        </rule>
        <rule name="React SPA Fallback" stopProcessing="true">
          <match url=".*" />
          <conditions logicalGrouping="MatchAll">
            <add input="{REQUEST_FILENAME}" matchType="IsFile" negate="true" />
            <add input="{REQUEST_FILENAME}" matchType="IsDirectory" negate="true" />
          </conditions>
          <action type="Rewrite" url="/index.html" />
        </rule>
      </rules>
    </rewrite>
    <httpErrors errorMode="DetailedLocalOnly" />
  </system.webServer>
</configuration>
```

- [ ] **Step 2: Add the backend service definition**

Create `deploy/windows/license-api-service.xml` with this content:

```xml
<service>
  <id>LicenseManagerApi</id>
  <name>License Manager API</name>
  <description>Go API for Enterprise License Manager</description>
  <executable>C:\Deploy\LicenseManager\backend\license-api.exe</executable>
  <workingdirectory>C:\Deploy\LicenseManager\backend</workingdirectory>
  <startmode>Automatic</startmode>
  <onfailure action="restart" delay="10 sec" />
  <onfailure action="restart" delay="30 sec" />
  <resetfailure>1 hour</resetfailure>
  <logpath>C:\Deploy\LicenseManager\backend\logs</logpath>
  <log mode="roll-by-size">
    <sizeThreshold>10485760</sizeThreshold>
    <keepFiles>5</keepFiles>
  </log>
</service>
```

- [ ] **Step 3: Build the web artifact and verify configuration copying**

Run on the development machine:

```powershell
Set-Location 'D:\DoAn\web'
npm.cmd test
npm.cmd run lint
npm.cmd run build
Test-Path '.\dist\web.config'
Select-String -Path '.\dist\web.config' -Pattern 'License Manager Backend','React SPA Fallback'
```

Expected: tests, lint, and build pass; `Test-Path` returns `True`; both IIS rules are found.

- [ ] **Step 4: Commit the deployment artifacts**

```powershell
Set-Location 'D:\DoAn'
git add web/public/web.config deploy/windows/license-api-service.xml
git commit -m "ops: add Windows IIS deployment configuration"
```

Expected: the commit contains only the two deployment configuration files.

### Task 4: Build a release bundle on the development machine

**Files:**
- Create, then package: `D:\DoAn\artifacts\windows-vps\`
- Consume: `backend/cmd/api`, `backend/cmd/migrate`, `backend/cmd/seed`, `backend/migrations`, `web/dist`

**Interfaces:**
- Consumes: tested Go and React source.
- Produces: `license-manager-windows-vps.zip` containing all server runtime files.

- [ ] **Step 1: Run the backend verification suite**

```powershell
Set-Location 'D:\DoAn\backend'
go test ./... -count=1
```

Expected: all enabled Go tests pass; PostgreSQL tests may report skipped only when `TEST_DATABASE_URL` is intentionally absent.

- [ ] **Step 2: Build three Windows executables**

```powershell
$artifactRoot = 'D:\DoAn\artifacts\windows-vps'
New-Item -ItemType Directory -Force -Path "$artifactRoot\backend","$artifactRoot\web","$artifactRoot\migrations","$artifactRoot\service" | Out-Null
Set-Location 'D:\DoAn\backend'
go build -trimpath -ldflags '-s -w' -o "$artifactRoot\backend\license-api.exe" ./cmd/api
go build -trimpath -ldflags '-s -w' -o "$artifactRoot\backend\license-migrate.exe" ./cmd/migrate
go build -trimpath -ldflags '-s -w' -o "$artifactRoot\backend\license-seed.exe" ./cmd/seed
```

Expected: all three EXE files exist and each has a non-zero length.

- [ ] **Step 3: Copy web, migrations, and service configuration**

```powershell
Copy-Item 'D:\DoAn\web\dist\*' "$artifactRoot\web" -Recurse -Force
Copy-Item 'D:\DoAn\backend\migrations\*' "$artifactRoot\migrations" -Recurse -Force
Copy-Item 'D:\DoAn\deploy\windows\license-api-service.xml' "$artifactRoot\service" -Force
New-Item -ItemType Directory -Force -Path "$artifactRoot\backend\logs","$artifactRoot\backups" | Out-Null
```

- [ ] **Step 4: Verify and package the bundle**

```powershell
$required = @(
  "$artifactRoot\backend\license-api.exe",
  "$artifactRoot\backend\license-migrate.exe",
  "$artifactRoot\backend\license-seed.exe",
  "$artifactRoot\web\index.html",
  "$artifactRoot\web\web.config",
  "$artifactRoot\migrations\001_initial_schema.sql",
  "$artifactRoot\service\license-api-service.xml"
)
$required | ForEach-Object { "$_ = $(Test-Path $_)" }
Compress-Archive -Path "$artifactRoot\*" -DestinationPath 'D:\DoAn\artifacts\license-manager-windows-vps.zip' -Force
Get-FileHash 'D:\DoAn\artifacts\license-manager-windows-vps.zip' -Algorithm SHA256
```

Expected: every required path reports `True`; record the SHA256 shown by PowerShell.

### Task 5: Transfer the bundle and initialize the VPS application

**Files:**
- Deploy: `C:\Deploy\LicenseManager\backend`, `web`, `migrations`, `service`, `backups`

**Interfaces:**
- Consumes: `license-manager-windows-vps.zip` from Task 4.
- Produces: an unpacked deployment tree and production process environment.

- [ ] **Step 1: Transfer and verify the bundle**

Copy the ZIP through the RDP shared drive to `C:\Deploy\license-manager-windows-vps.zip`, then run on the VPS:

```powershell
Get-FileHash 'C:\Deploy\license-manager-windows-vps.zip' -Algorithm SHA256
Expand-Archive 'C:\Deploy\license-manager-windows-vps.zip' 'C:\Deploy\LicenseManager' -Force
Get-ChildItem 'C:\Deploy\LicenseManager'
```

Expected: the VPS hash matches Task 4 and the five deployment directories appear.

- [ ] **Step 2: Generate production application secrets**

Run on the VPS and do not print the resulting values:

```powershell
$jwtBytes = New-Object byte[] 48
$licenseKeyBytes = New-Object byte[] 32
$rng = [Security.Cryptography.RandomNumberGenerator]::Create()
$rng.GetBytes($jwtBytes)
$rng.GetBytes($licenseKeyBytes)
$env:JWT_SECRET = [Convert]::ToBase64String($jwtBytes)
$env:LICENSE_ENCRYPTION_KEY = [Convert]::ToBase64String($licenseKeyBytes)
$rng.Dispose()
[Convert]::FromBase64String($env:LICENSE_ENCRYPTION_KEY).Length
```

Expected: `32`.

- [ ] **Step 3: Build the database URL without printing the password**

Run on the VPS. Enter the `license_admin` password created in Task 2 when prompted:

```powershell
$secureDatabasePassword = Read-Host 'Password for license_admin' -AsSecureString
$passwordPointer = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($secureDatabasePassword)
try {
  $databasePassword = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($passwordPointer)
  $encodedDatabasePassword = [Uri]::EscapeDataString($databasePassword)
  $env:DATABASE_URL = "postgres://license_admin:$encodedDatabasePassword@127.0.0.1:5432/license_manager?sslmode=disable"
} finally {
  [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($passwordPointer)
  $databasePassword = $null
}
```

- [ ] **Step 4: Set and persist the backend environment for the Windows service**

```powershell
$env:APP_ENV = 'production'
$env:HTTP_ADDRESS = '127.0.0.1:8080'
$env:STORAGE_DRIVER = 'postgres'
$env:SHUTDOWN_TIMEOUT = '10s'
$env:JWT_ISSUER = 'license-manager'
$env:ACCESS_TOKEN_TTL = '15m'
$env:REFRESH_TOKEN_TTL = '168h'
[Environment]::SetEnvironmentVariable('APP_ENV',$env:APP_ENV,'Machine')
[Environment]::SetEnvironmentVariable('HTTP_ADDRESS',$env:HTTP_ADDRESS,'Machine')
[Environment]::SetEnvironmentVariable('STORAGE_DRIVER',$env:STORAGE_DRIVER,'Machine')
[Environment]::SetEnvironmentVariable('DATABASE_URL',$env:DATABASE_URL,'Machine')
[Environment]::SetEnvironmentVariable('JWT_SECRET',$env:JWT_SECRET,'Machine')
[Environment]::SetEnvironmentVariable('LICENSE_ENCRYPTION_KEY',$env:LICENSE_ENCRYPTION_KEY,'Machine')
[Environment]::SetEnvironmentVariable('SHUTDOWN_TIMEOUT',$env:SHUTDOWN_TIMEOUT,'Machine')
[Environment]::SetEnvironmentVariable('JWT_ISSUER',$env:JWT_ISSUER,'Machine')
[Environment]::SetEnvironmentVariable('ACCESS_TOKEN_TTL',$env:ACCESS_TOKEN_TTL,'Machine')
[Environment]::SetEnvironmentVariable('REFRESH_TOKEN_TTL',$env:REFRESH_TOKEN_TTL,'Machine')
```

Expected: the commands return without displaying secret values.

- [ ] **Step 5: Back up the encryption key outside the web root**

Store the current `LICENSE_ENCRYPTION_KEY` in the user's password manager or another encrypted location. Do not place it in `C:\Deploy\LicenseManager\web`, Git, a screenshot, or chat.

### Task 6: Run migrations, seed demo data, and test the backend EXE

**Files:**
- Consume: `C:\Deploy\LicenseManager\migrations\*.sql`
- Execute: the three backend EXE files.

**Interfaces:**
- Consumes: production environment variables and fresh PostgreSQL database.
- Produces: migrated schema, demo records, and a healthy API at `127.0.0.1:8080`.

- [ ] **Step 1: Apply migrations**

```powershell
Set-Location 'C:\Deploy\LicenseManager'
.\backend\license-migrate.exe up
.\backend\license-migrate.exe status
```

Expected: migration succeeds and every repository migration is reported as applied.

- [ ] **Step 2: Seed the production demo database**

Set private admin credentials only in this PowerShell process, then seed:

```powershell
$env:DEV_ADMIN_EMAIL = 'admin@local.test'
$env:DEV_ADMIN_PASSWORD = Read-Host 'Create the demo admin password'
$env:SEED_DEMO_DATA = 'true'
.\backend\license-seed.exe
$env:DEV_ADMIN_PASSWORD = $null
```

Expected: the admin and demo data are created without duplicate-key or migration errors.

- [ ] **Step 3: Start the backend interactively**

```powershell
Set-Location 'C:\Deploy\LicenseManager\backend'
.\license-api.exe
```

Expected: log output includes `http server started`, `127.0.0.1:8080`, and PostgreSQL storage. Keep this window open for Task 7.

- [ ] **Step 4: Verify health from a second VPS PowerShell window**

```powershell
Invoke-RestMethod 'http://127.0.0.1:8080/health/live'
Invoke-RestMethod 'http://127.0.0.1:8080/health/ready'
```

Expected: both endpoints report healthy/ready state.

### Task 7: Install IIS, proxy API traffic, and serve React

**Files:**
- Consume: `C:\Deploy\LicenseManager\web\web.config`
- Configure: IIS Default Web Site.

**Interfaces:**
- Consumes: healthy backend on `127.0.0.1:8080`.
- Produces: HTTP web/API access through IIS on port 80.

- [ ] **Step 1: Install IIS features**

Run in elevated PowerShell on the VPS:

```powershell
Install-WindowsFeature Web-Server,Web-Static-Content,Web-Default-Doc,Web-Http-Errors,Web-Http-Logging,Web-Mgmt-Console -IncludeManagementTools
Import-Module WebAdministration
Get-Website
```

Expected: IIS installation succeeds and `Default Web Site` is listed.

- [ ] **Step 2: Install URL Rewrite and ARR**

Download and install Microsoft IIS URL Rewrite 2 and Application Request Routing 3. Open IIS Manager, select the server node, open **Application Request Routing Cache**, choose **Server Proxy Settings**, enable **Enable proxy**, and click **Apply**.

- [ ] **Step 3: Point IIS to the React build**

```powershell
Import-Module WebAdministration
Set-ItemProperty 'IIS:\Sites\Default Web Site' -Name physicalPath -Value 'C:\Deploy\LicenseManager\web'
Restart-WebAppPool 'DefaultAppPool'
Restart-WebItem 'IIS:\Sites\Default Web Site'
```

Expected: `Invoke-WebRequest http://127.0.0.1` returns HTTP 200.

- [ ] **Step 4: Verify IIS proxy and SPA fallback locally**

```powershell
(Invoke-WebRequest -UseBasicParsing 'http://127.0.0.1/health/ready').StatusCode
(Invoke-WebRequest -UseBasicParsing 'http://127.0.0.1/').StatusCode
(Invoke-WebRequest -UseBasicParsing 'http://127.0.0.1/licenses').StatusCode
```

Expected: all three return `200`; the last request is served by React's `index.html` rather than IIS 404.

- [ ] **Step 5: Verify from an external network**

Disable Wi-Fi on a phone and browse to:

```text
http://tranhieu1834.duckdns.org/health/ready
http://tranhieu1834.duckdns.org
```

Expected: health and web are reachable through mobile data. If local access works but mobile data fails, fix router forwarding or ISP filtering before continuing.

### Task 8: Enable a trusted HTTPS certificate

**Files:**
- Managed by win-acme: certificate and scheduled renewal task.

**Interfaces:**
- Consumes: reachable domain on TCP 80 and IIS site binding.
- Produces: valid HTTPS binding for `tranhieu1834.duckdns.org` on TCP 443.

- [ ] **Step 1: Install win-acme**

Download the current 64-bit pluggable win-acme release from its official site, extract it to `C:\Tools\win-acme`, and run `wacs.exe` as Administrator.

- [ ] **Step 2: Request the IIS certificate**

In win-acme, choose to create a certificate for an IIS site, select `Default Web Site`, use the binding hostname `tranhieu1834.duckdns.org`, accept the IIS installation step, and allow win-acme to create its scheduled renewal task.

- [ ] **Step 3: Require HTTPS after certificate issuance**

In IIS Manager, open **URL Rewrite** for `Default Web Site`, add a **Blank rule**, and set these exact values:

```text
Name: Redirect HTTP to HTTPS
Requested URL: Matches the Pattern
Using: Regular Expressions
Pattern: (.*)
Condition input: {HTTPS}
Condition pattern: ^OFF$
Action type: Redirect
Redirect URL: https://{HTTP_HOST}/{R:1}
Append query string: enabled
Redirect type: Permanent (301)
Stop processing: enabled
```

Move this rule above `License Manager Backend` and `React SPA Fallback`, then click **Apply**.

- [ ] **Step 4: Verify HTTPS externally**

Run from the development machine:

```powershell
(Invoke-WebRequest -UseBasicParsing 'https://tranhieu1834.duckdns.org/health/ready').StatusCode
(Invoke-WebRequest -UseBasicParsing 'http://tranhieu1834.duckdns.org' -MaximumRedirection 0 -ErrorAction SilentlyContinue).StatusCode
```

Expected: HTTPS returns `200`; HTTP returns a redirect status in the 300 range.

### Task 9: Convert the backend EXE to an automatic Windows Service

**Files:**
- Create on VPS: `C:\Deploy\LicenseManager\service\license-api-service.exe`
- Consume: `C:\Deploy\LicenseManager\service\license-api-service.xml`

**Interfaces:**
- Consumes: machine-level environment variables from Task 5.
- Produces: automatic service `LicenseManagerApi` with restart-on-failure logging.

- [ ] **Step 1: Stop the interactive backend**

Press `Ctrl+C` in the PowerShell window running `license-api.exe`, then verify:

```powershell
Test-NetConnection 127.0.0.1 -Port 8080 -InformationLevel Quiet
```

Expected: `False`.

- [ ] **Step 2: Install WinSW**

Download the stable x64 WinSW executable from the official WinSW GitHub releases page. Copy it to `C:\Deploy\LicenseManager\service` and rename it to `license-api-service.exe`. Keep `license-api-service.exe` and `license-api-service.xml` in the same directory.

- [ ] **Step 3: Restrict access to service configuration and install the service**

```powershell
$servicePath = 'C:\Deploy\LicenseManager\service'
icacls $servicePath /inheritance:r
icacls $servicePath /grant:r 'Administrators:(OI)(CI)F' 'SYSTEM:(OI)(CI)F'
Set-Location $servicePath
.\license-api-service.exe install
.\license-api-service.exe start
Get-Service LicenseManagerApi | Select-Object Name,Status,StartType
```

Expected: service status is `Running` and start type is automatic.

- [ ] **Step 4: Verify the service through IIS**

```powershell
Invoke-RestMethod 'http://127.0.0.1:8080/health/ready'
Invoke-RestMethod 'https://tranhieu1834.duckdns.org/health/ready'
```

Expected: both calls succeed.

- [ ] **Step 5: Reboot acceptance test**

```powershell
Restart-Computer
```

After reconnecting through RDP:

```powershell
Get-Service postgresql*,W3SVC,LicenseManagerApi | Select-Object Name,Status,StartType
Invoke-RestMethod 'https://tranhieu1834.duckdns.org/health/ready'
```

Expected: PostgreSQL, IIS, and License Manager API are running automatically; public readiness succeeds.

### Task 10: Point Android to the deployed API and complete acceptance testing

**Files:**
- Modify: `.worktrees/feature-android-employee-app/android/app/build.gradle.kts`
- Test: Android unit tests and emulator/device login.

**Interfaces:**
- Consumes: trusted HTTPS API at `https://tranhieu1834.duckdns.org/api/v1/`.
- Produces: Android builds that use the deployed API without LAN IP addresses or cleartext HTTP.

- [ ] **Step 1: Replace Android API base URLs**

Set both the default/release and debug `API_BASE_URL` values in `android/app/build.gradle.kts` to:

```kotlin
"\"https://tranhieu1834.duckdns.org/api/v1/\""
```

- [ ] **Step 2: Run Android unit tests**

```powershell
Set-Location 'D:\DoAn\.worktrees\feature-android-employee-app\android'
.\gradlew.bat testDebugUnitTest
```

Expected: `BUILD SUCCESSFUL`.

- [ ] **Step 3: Run end-to-end user checks**

Install the debug app on the emulator or Android device and verify:

```text
Employee login
Dashboard refresh
Assigned licenses and controlled activation-key access
Assigned devices
Create/cancel a license request
Create a maintenance request
Receive and mark notifications as read
View the personal profile
```

Expected: every action works without using the VPS LAN IP or port 8080.

- [ ] **Step 4: Verify public port exposure**

Run from a machine outside the VPS LAN:

```powershell
Test-NetConnection tranhieu1834.duckdns.org -Port 80 -InformationLevel Quiet
Test-NetConnection tranhieu1834.duckdns.org -Port 443 -InformationLevel Quiet
Test-NetConnection tranhieu1834.duckdns.org -Port 5432 -InformationLevel Quiet
Test-NetConnection tranhieu1834.duckdns.org -Port 8080 -InformationLevel Quiet
```

Expected: 80 and 443 return `True`; 5432 and 8080 return `False`.

- [ ] **Step 5: Commit the Android endpoint change**

```powershell
Set-Location 'D:\DoAn\.worktrees\feature-android-employee-app'
git add android/app/build.gradle.kts
git commit -m "chore: point Android app to deployed API"
```

Expected: only the Android build configuration is committed.

### Task 11: Document backup and rollback evidence

**Files:**
- Create on VPS: `C:\Deploy\LicenseManager\backups\license_manager_initial.backup`
- Create on VPS: `C:\Deploy\LicenseManager\backups\acceptance.txt`

**Interfaces:**
- Consumes: healthy deployed database and endpoints.
- Produces: restorable baseline plus acceptance evidence for the thesis demo.

- [ ] **Step 1: Create the initial PostgreSQL backup**

```powershell
& 'C:\Program Files\PostgreSQL\15\bin\pg_dump.exe' $env:DATABASE_URL -Fc -f 'C:\Deploy\LicenseManager\backups\license_manager_initial.backup'
Get-Item 'C:\Deploy\LicenseManager\backups\license_manager_initial.backup' | Select-Object FullName,Length,LastWriteTime
```

Expected: backup file exists with non-zero length.

- [ ] **Step 2: Capture non-secret acceptance evidence**

```powershell
$evidence = 'C:\Deploy\LicenseManager\backups\acceptance.txt'
Get-Date | Out-File $evidence
Get-Service postgresql*,W3SVC,LicenseManagerApi | Select-Object Name,Status,StartType | Out-File $evidence -Append
Invoke-RestMethod 'https://tranhieu1834.duckdns.org/health/live' | Out-File $evidence -Append
Invoke-RestMethod 'https://tranhieu1834.duckdns.org/health/ready' | Out-File $evidence -Append
Get-FileHash 'C:\Deploy\license-manager-windows-vps.zip' -Algorithm SHA256 | Out-File $evidence -Append
```

Expected: the evidence file contains no passwords, tokens, database URLs, or encryption keys.

- [ ] **Step 3: Record the rollback procedure**

For an application-only failure: stop `LicenseManagerApi`, restore the previous backend/web artifact directories from `backups`, then restart the service. For a schema/data failure: stop the API, recreate `license_manager`, restore the most recent `.backup` using `pg_restore`, restore the matching `LICENSE_ENCRYPTION_KEY`, and start the API.
