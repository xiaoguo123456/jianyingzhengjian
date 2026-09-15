#!/usr/bin/env bash
# End-to-end smoke test against a running api + worker (mock providers, local storage).
set -euo pipefail
API=${API:-http://localhost:8080}
J() { curl -sS "$@"; }
echo "== login"
TOKEN=$(J -X POST $API/v1/auth/login -H 'Content-Type: application/json' -d '{"provider":"h5_dev","code":"smoke"}' | python3 -c 'import sys,json; print(json.load(sys.stdin)["data"]["token"])')
AUTH="Authorization: Bearer $TOKEN"
echo "== me"; ME=$(J $API/v1/me -H "$AUTH"); echo "$ME" | python3 -c 'import sys,json; d=json.load(sys.stdin)["data"]; print(d["user"]["id"], d["credits"])'
USER_ID=$(echo "$ME" | python3 -c 'import sys,json; print(json.load(sys.stdin)["data"]["user"]["id"])')
echo "== admin login + top up credits (keeps the run repeatable)"
ADMIN_TOKEN=$(J -X POST $API/admin/v1/auth/login -H 'Content-Type: application/json' -d '{"username":"admin","password":"admin123"}' | python3 -c 'import sys,json; print(json.load(sys.stdin)["data"]["token"])')
# set the bonus balance to exactly 2 so the drain step below is deterministic
BONUS=$(J "$API/admin/v1/users/$USER_ID" -H "Authorization: Bearer $ADMIN_TOKEN" | python3 -c 'import sys,json; d=json.load(sys.stdin)["data"]; print((d.get("account") or {}).get("bonus_credits", 0))')
if [ "$BONUS" -gt 0 ]; then
  J -X POST $API/admin/v1/users/$USER_ID/credits -H "Authorization: Bearer $ADMIN_TOKEN" -H 'Content-Type: application/json' -d "{\"delta\":-$BONUS,\"note\":\"smoke reset\"}" > /dev/null
fi
J -X POST $API/admin/v1/users/$USER_ID/credits -H "Authorization: Bearer $ADMIN_TOKEN" -H 'Content-Type: application/json' -d '{"delta":2,"note":"smoke test"}' | python3 -c 'import sys,json; d=json.load(sys.stdin); print("   bonus set to:", d["data"]["balance_bonus_after"] if d["code"]=="OK" else d)'
echo "== privacy"; J -X POST $API/v1/me/privacy-agree -H "$AUTH" -H 'Content-Type: application/json' -d '{"version":"2026-09-01"}' >/dev/null
echo "== home idphoto"; J $API/v1/home/idphoto -H "$AUTH" | python3 -c 'import sys,json; d=json.load(sys.stdin)["data"]; print("banner:", d["banner"]["title"].replace("\n"," "), "| hot specs:", [s["name"] for s in d["hot_specs"]])'
echo "== home portrait"; J $API/v1/home/portrait -H "$AUTH" | python3 -c 'import sys,json; d=json.load(sys.stdin)["data"]; print("templates:", [t["name"] for t in d["hot_templates"]], "| cover:", d["hot_templates"][0]["cover_url"][:60])'
echo "== upload"; UP=$(J -X POST $API/v1/photos -H "$AUTH" -F module=idphoto -F file=@testdata/portrait.jpg); echo "$UP" | python3 -c 'import sys,json; d=json.load(sys.stdin); print(d["code"], d["data"]["check"] if d["code"]=="OK" else d)'
PHOTO=$(echo "$UP" | python3 -c 'import sys,json; print(json.load(sys.stdin)["data"]["photo"]["id"])')
echo "== create free idphoto task"
T=$(J -X POST $API/v1/tasks -H "$AUTH" -H 'Content-Type: application/json' -H "Idempotency-Key: smoke-$(date +%s)" -d "{\"kind\":\"idphoto\",\"spec_id\":\"sp_1inch\",\"photo_id\":\"$PHOTO\",\"params\":{\"bg\":\"#438EDB\",\"clothing\":\"keep\",\"beauty\":\"natural\"},\"notify\":false}")
echo "$T" | python3 -c 'import sys,json; d=json.load(sys.stdin); print(d["code"], d["data"]["task"]["status"], "uses_gen:", d["data"]["task"]["uses_genmodel"], "credits:", d["data"]["credits"]["total"])'
TASK=$(echo "$T" | python3 -c 'import sys,json; print(json.load(sys.stdin)["data"]["task"]["id"])')
for i in $(seq 1 30); do
  S=$(J $API/v1/tasks/$TASK -H "$AUTH"); ST=$(echo "$S" | python3 -c 'import sys,json; d=json.load(sys.stdin)["data"]; print(d["status"], d["stage"])')
  echo "   poll: $ST"; case "$ST" in success*|failed*) break;; esac; sleep 1
done
echo "$S" | python3 -c 'import sys,json; d=json.load(sys.stdin)["data"]; w=d.get("work"); print("work:", w and (w["id"], w["width"], w["height"], w["url"][:70])); print("error:", d.get("error"))'
WORK=$(echo "$S" | python3 -c 'import sys,json; d=json.load(sys.stdin)["data"]; print(d["work"]["id"] if d.get("work") else "")')
if [ -n "$WORK" ]; then
  echo "== download url"; URL=$(J $API/v1/works/$WORK/download -H "$AUTH" | python3 -c 'import sys,json; print(json.load(sys.stdin)["data"]["url"])'); curl -sS -o /tmp/smoke_work.png "$URL"; file /tmp/smoke_work.png
  echo "== recolor"; J -X POST $API/v1/works/$WORK/recolor -H "$AUTH" -H 'Content-Type: application/json' -d '{"bg":"#FFFFFF"}' | python3 -c 'import sys,json; d=json.load(sys.stdin); print(d["code"], d["data"]["work"]["meta"] if d["code"]=="OK" else d)'
fi
echo "== template task (gen model, consumes credit)"
T2=$(J -X POST $API/v1/tasks -H "$AUTH" -H 'Content-Type: application/json' -H "Idempotency-Key: smoke2-$(date +%s)" -d "{\"kind\":\"template\",\"template_id\":\"t_interview\",\"photo_id\":\"$PHOTO\",\"params\":{},\"notify\":false}")
echo "$T2" | python3 -c 'import sys,json; d=json.load(sys.stdin); print(d["code"], d.get("message"), d["data"]["task"]["status"] if d["code"]=="OK" else d["data"])'
TASK2=$(echo "$T2" | python3 -c 'import sys,json; d=json.load(sys.stdin); print(d["data"]["task"]["id"] if d["code"]=="OK" else "")')
if [ -n "$TASK2" ]; then for i in $(seq 1 40); do S2=$(J $API/v1/tasks/$TASK2 -H "$AUTH"); ST=$(echo "$S2" | python3 -c 'import sys,json; d=json.load(sys.stdin)["data"]; print(d["status"], d["stage"])'); echo "   poll: $ST"; case "$ST" in success*|failed*) break;; esac; sleep 1; done; echo "$S2" | python3 -c 'import sys,json; d=json.load(sys.stdin)["data"]; print("work:", d["work"] and d["work"]["id"], "ai_label:", d["work"] and d["work"]["ai_label"], "error:", d["error"])'; fi
echo "== drain credits, then expect NO_CREDITS"
CODE=""
for i in $(seq 1 8); do
  R=$(J -X POST $API/v1/tasks -H "$AUTH" -H 'Content-Type: application/json' -H "Idempotency-Key: drain-$i-$(date +%s)" -d "{\"kind\":\"template\",\"template_id\":\"t_autumn\",\"photo_id\":\"$PHOTO\",\"params\":{},\"notify\":false}")
  CODE=$(echo "$R" | python3 -c 'import sys,json; print(json.load(sys.stdin)["code"])')
  echo "   attempt $i -> $CODE"
  [ "$CODE" = "NO_CREDITS" ] && break
done
[ "$CODE" = "NO_CREDITS" ] && echo "   credit gate works" || { echo "   FAIL: never hit NO_CREDITS"; exit 1; }
echo "== ad session (ads disabled -> 409 expected)"
J -X POST $API/v1/ads/sessions -H "$AUTH" -o /dev/null -w "   ads/sessions: %{http_code}\n"
echo "== share + summary"
J -X POST $API/v1/shares -H "$AUTH" -H 'Content-Type: application/json' -d "{\"type\":\"work\",\"work_id\":\"$WORK\",\"surface\":\"smoke\"}" | python3 -c 'import sys,json; d=json.load(sys.stdin); print(d["code"], d["data"]["share"]["path"] if d["code"]=="OK" else d)'
J $API/v1/works/summary -H "$AUTH" | python3 -c 'import sys,json; d=json.load(sys.stdin)["data"]; print("summary:", d["total"], d["by_module"])'
echo "== admin stats"
J $API/admin/v1/stats/templates -H "Authorization: Bearer $ADMIN_TOKEN" | python3 -c 'import sys,json; d=json.load(sys.stdin); print("   template stats rows:", len(d["data"]))'
J $API/admin/v1/stats/funnel -H "Authorization: Bearer $ADMIN_TOKEN" | python3 -c 'import sys,json; d=json.load(sys.stdin)["data"]; print("   funnel:", {r["name"]: r["count"] for r in d if r["count"]})'
J $API/admin/v1/configs -H "Authorization: Bearer $ADMIN_TOKEN" | python3 -c 'import sys,json; d=json.load(sys.stdin)["data"]; print("   configs:", len(d), "keys")'
echo "SMOKE DONE"
