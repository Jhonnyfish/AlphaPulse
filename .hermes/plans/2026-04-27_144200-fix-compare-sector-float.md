# Fix `/api/compare/sector` 500 Error — EastMoney `"-"` String Handling

## Goal
Fix the `/api/compare/sector` endpoint returning HTTP 500 when EastMoney returns `"-"` string values for numeric fields.

## Root Cause
EastMoney's sector members API returns `"-"` (string) for some numeric fields (f3 change_pct, f22 amount) when data is unavailable. The `sectorMemberFields` struct in `eastmoney.go` uses `float64` for these fields, causing `json.Unmarshal` to fail.

The error flow in `parseSectorMembersDiff` (line 1368):
1. `json.Unmarshal(raw, &list)` fails because some array elements have `"-"` for float64 fields
2. Fallback `json.Unmarshal(raw, &dict)` also fails because the data IS an array, not a map
3. Error propagates up → HTTP 500

## Affected Fields
- `f3` (ChangePct) — can be `"-"` 
- `f22` (Amount) — can be `"-"`

## Proposed Fix
Add a `flexFloat64` custom type that unmarshals both numeric values and `"-"` strings (converting `"-"` to 0.0):

```go
// flexFloat64 handles JSON values that can be either a number or the string "-".
type flexFloat64 float64

func (f *flexFloat64) UnmarshalJSON(data []byte) error {
    // Try numeric first
    var v float64
    if err := json.Unmarshal(data, &v); err == nil {
        *f = flexFloat64(v)
        return nil
    }
    // String value — treat "-" or non-numeric as 0
    var s string
    if err := json.Unmarshal(data, &s); err == nil {
        v, err := strconv.ParseFloat(s, 64)
        if err != nil {
            *f = 0
            return nil
        }
        *f = flexFloat64(v)
        return nil
    }
    *f = 0
    return nil
}
```

Then update `sectorMemberFields`:
```go
type sectorMemberFields struct {
    Code      string     `json:"f12"`
    Name      string     `json:"f14"`
    ChangePct flexFloat64 `json:"f3"`
    PE        flexFloat64 `json:"f9"`
    PB        flexFloat64 `json:"f23"`
    Amount    flexFloat64 `json:"f22"`
}
```

Update downstream code to convert `flexFloat64` to `float64` where needed (casting or helper).

## Files to Change
- `internal/services/eastmoney.go`
  - Add `flexFloat64` type + UnmarshalJSON method (near line 1359)
  - Update `sectorMemberFields` struct (line 1359-1366) to use `flexFloat64`
  - Update `FetchSectorMembers` (line 450-465) where it copies to `models.SectorMember` — cast `flexFloat64` to `float64`

## Verification
1. `cd /home/finn/alphapulse && go build ./...`
2. `go vet ./...`
3. Restart server: kill and restart `./alphapulse`
4. Test: `curl -s -H "Authorization: Bearer $TOKEN" "http://localhost:8899/api/compare/sector?code=600519"` → should return 200 with sector data
