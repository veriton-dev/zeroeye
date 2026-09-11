package aitools

import (
	"testing"
)

func TestReadMarketDataTool(t *testing.T) {
	r := DefaultTools()
	specs := r.List()
	if len(specs) != 1 || specs[0].Name != "read_market_data" {
		t.Fatalf("expected read_market_data registered, got %#v", specs)
	}
	res := r.Invoke("read_market_data", map[string]any{"symbol": "btc-usd"})
	if !res.OK {
		t.Fatalf("invoke failed: %s", res.Error)
	}
	if res.Data["symbol"] != "BTC-USD" {
		t.Fatalf("symbol normalize failed: %#v", res.Data)
	}
	for _, k := range []string{"bid", "ask", "mid", "volume", "ts"} {
		if _, ok := res.Data[k]; !ok {
			t.Fatalf("missing field %s in %#v", k, res.Data)
		}
	}
	res2 := r.Invoke("read_market_data", map[string]any{
		"symbol": "ETH-USD",
		"fields": []any{"mid", "volume"},
	})
	if !res2.OK {
		t.Fatalf("filtered invoke failed: %s", res2.Error)
	}
	if _, ok := res2.Data["bid"]; ok {
		t.Fatalf("bid should be filtered out: %#v", res2.Data)
	}
	if res2.Data["mid"] == nil {
		t.Fatalf("mid missing: %#v", res2.Data)
	}
	bad := r.Invoke("nope", nil)
	if bad.OK {
		t.Fatal("unknown tool should fail")
	}
	empty := r.Invoke("read_market_data", map[string]any{"symbol": ""})
	if empty.OK {
		t.Fatal("empty symbol should fail")
	}
}
