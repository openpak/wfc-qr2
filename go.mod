module openpak/wfc-qr2

go 1.26.0

require openpak/wfc-common v0.0.0

replace openpak/wfc-common => ../wfc-common

require (
	github.com/linkdata/deadlock v0.5.5
	github.com/logrusorgru/aurora/v3 v3.0.0
	gvisor.dev/gvisor v0.0.0-20250512220230-2268d0cbb0f5
)

require (
	github.com/jackc/pgerrcode v0.0.0-20250907135507-afb5586c32a6 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.11.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/petermattis/goid v0.0.0-20250813065127-a731cc31b4fe // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/text v0.41.0 // indirect
)
