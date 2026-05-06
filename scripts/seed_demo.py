#!/usr/bin/env python3
"""Seed demo data for the baby-tracker dev server.

Creates one child plus four trackers covering all chartable field types
(number, duration, select, multiselect, boolean, longtext) and 1–10 entries
per tracker per day across the last N days (default 100).

Usage:
    python3 scripts/seed_demo.py                              # admin/admin1234 @ :8080
    BT_URL=http://localhost:8080 \\
    BT_USER=admin BT_PASS=admin1234 \\
    python3 scripts/seed_demo.py --days 100 --child-name Lina

Re-running appends fresh trackers/entries — wipe ``app.db`` first if you
want a clean slate.
"""

from __future__ import annotations

import argparse
import datetime as dt
import http.cookiejar
import json
import os
import random
import sys
import urllib.error
import urllib.request


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--base-url", default=os.environ.get("BT_URL", "http://localhost:8080"))
    ap.add_argument("--username", default=os.environ.get("BT_USER", "admin"))
    ap.add_argument("--password", default=os.environ.get("BT_PASS", "admin1234"))
    ap.add_argument("--days", type=int, default=100, help="how many days back to seed")
    ap.add_argument("--profile-name", default="Lina")
    ap.add_argument("--profile-description", default=None)
    ap.add_argument("--seed", type=int, default=None, help="random seed for repeatability")
    args = ap.parse_args()

    if args.seed is not None:
        random.seed(args.seed)

    api = APIClient(args.base_url)
    api.login(args.username, args.password)
    print(f"logged in as {args.username} @ {args.base_url}")

    profile = api.create_profile(args.profile_name, description=args.profile_description)
    print(f"profile created: id={profile['id']} name={profile['name']!r}")

    trackers = []
    for tdef in tracker_defs():
        t = api.create_tracker(profile["id"], tdef)
        trackers.append(t)
        print(f"tracker created: id={t['id']} name={t['name']!r}")

    seed_entries(api, trackers, args.days)


# ---------------------------------------------------------------------------
# API client
# ---------------------------------------------------------------------------


class APIClient:
    def __init__(self, base_url: str) -> None:
        self.base_url = base_url.rstrip("/")
        self.cookies = http.cookiejar.CookieJar()
        self.opener = urllib.request.build_opener(
            urllib.request.HTTPCookieProcessor(self.cookies)
        )

    def request(self, method: str, path: str, body=None):
        url = self.base_url + path
        data = json.dumps(body).encode("utf-8") if body is not None else None
        req = urllib.request.Request(url, data=data, method=method)
        if data is not None:
            req.add_header("Content-Type", "application/json")
        try:
            resp = self.opener.open(req, timeout=30)
        except urllib.error.HTTPError as e:
            payload = e.read().decode("utf-8", "replace")
            sys.exit(f"{method} {path} -> {e.code}: {payload}")
        except urllib.error.URLError as e:
            sys.exit(f"{method} {path} -> {e.reason}")
        raw = resp.read()
        if not raw:
            return None
        return json.loads(raw)

    def login(self, username: str, password: str) -> None:
        self.request("POST", "/api/auth/login", {"username": username, "password": password})

    def create_profile(self, name: str, description: str | None = None, avatar_url: str | None = None):
        out = self.request(
            "POST",
            "/api/profiles",
            {"name": name, "description": description, "avatar_url": avatar_url},
        )
        return out["profile"]

    def create_tracker(self, profile_id: int, payload):
        out = self.request("POST", f"/api/profiles/{profile_id}/trackers", payload)
        return out["tracker"]

    def create_entry(self, tracker_id: int, data) -> None:
        self.request(
            "POST",
            f"/api/trackers/{tracker_id}/entries",
            {"data": data},
        )


# ---------------------------------------------------------------------------
# Tracker schema definitions — covers every chartable field type
# ---------------------------------------------------------------------------


def tracker_defs():
    return [
        {
            "name": "Feeding",
            "icon": "milk",
            "color": "#0ea5e9",
            "description": "Bottle, breast, or solids.",
            "schema_json": {
                "fields": [
                    {
                        "key": "occurred_at",
                        "label": {"en": "When", "uk": "Коли"},
                        "type": "datetime",
                        "required": True,
                        "isPrimaryTime": True,
                    },
                    {
                        "key": "amount",
                        "label": {"en": "Amount", "uk": "Кількість"},
                        "type": "number",
                        "unit": "ml",
                        "min": 0,
                        "required": True,
                    },
                    {
                        "key": "type",
                        "label": {"en": "Type", "uk": "Тип"},
                        "type": "select",
                        "options": [
                            {"value": "bottle", "label": {"en": "Bottle", "uk": "Пляшка"}},
                            {"value": "breast", "label": {"en": "Breast", "uk": "Груди"}},
                            {"value": "solid", "label": {"en": "Solid food", "uk": "Прикорм"}},
                        ],
                    },
                    {
                        "key": "note",
                        "label": {"en": "Note", "uk": "Примітка"},
                        "type": "longtext",
                    },
                ],
            },
        },
        {
            "name": "Sleep",
            "icon": "moon",
            "color": "#6366f1",
            "description": "Naps and night sleep.",
            "schema_json": {
                "fields": [
                    {
                        "key": "occurred_at",
                        "label": {"en": "Started", "uk": "Початок"},
                        "type": "datetime",
                        "required": True,
                        "isPrimaryTime": True,
                    },
                    {
                        "key": "length",
                        "label": {"en": "Duration", "uk": "Тривалість"},
                        "type": "duration",
                        "required": True,
                    },
                    {
                        "key": "where",
                        "label": {"en": "Where", "uk": "Де"},
                        "type": "select",
                        "options": [
                            {"value": "crib", "label": {"en": "Crib", "uk": "Ліжко"}},
                            {"value": "stroller", "label": {"en": "Stroller", "uk": "Коляска"}},
                            {"value": "parents_bed", "label": {"en": "Parents' bed", "uk": "У батьків"}},
                            {"value": "car", "label": {"en": "Car", "uk": "Авто"}},
                        ],
                    },
                    {
                        "key": "woke_up",
                        "label": {"en": "Woke during", "uk": "Прокидався"},
                        "type": "boolean",
                    },
                ],
            },
        },
        {
            "name": "Diaper",
            "icon": "droplet",
            "color": "#10b981",
            "description": "Diaper changes.",
            "schema_json": {
                "fields": [
                    {
                        "key": "occurred_at",
                        "label": {"en": "When", "uk": "Коли"},
                        "type": "datetime",
                        "required": True,
                        "isPrimaryTime": True,
                    },
                    {
                        "key": "type",
                        "label": {"en": "Type", "uk": "Тип"},
                        "type": "select",
                        "options": [
                            {"value": "wet", "label": {"en": "Wet", "uk": "Мокрий"}},
                            {"value": "dirty", "label": {"en": "Dirty", "uk": "Брудний"}},
                            {"value": "both", "label": {"en": "Both", "uk": "Обидва"}},
                            {"value": "dry", "label": {"en": "Dry", "uk": "Сухий"}},
                        ],
                    },
                    {
                        "key": "rash",
                        "label": {"en": "Rash", "uk": "Подразнення"},
                        "type": "boolean",
                    },
                ],
            },
        },
        {
            "name": "Activity",
            "icon": "activity",
            "color": "#f59e0b",
            "description": "Tummy time, walks, bath, play.",
            "schema_json": {
                "fields": [
                    {
                        "key": "occurred_at",
                        "label": {"en": "When", "uk": "Коли"},
                        "type": "datetime",
                        "required": True,
                        "isPrimaryTime": True,
                    },
                    {
                        "key": "kinds",
                        "label": {"en": "Kinds", "uk": "Види"},
                        "type": "multiselect",
                        "options": [
                            {"value": "tummy_time", "label": {"en": "Tummy time", "uk": "На животі"}},
                            {"value": "walk", "label": {"en": "Walk", "uk": "Прогулянка"}},
                            {"value": "bath", "label": {"en": "Bath", "uk": "Купання"}},
                            {"value": "play", "label": {"en": "Play", "uk": "Гра"}},
                            {"value": "reading", "label": {"en": "Reading", "uk": "Читання"}},
                        ],
                    },
                    {
                        "key": "duration_min",
                        "label": {"en": "Duration", "uk": "Тривалість"},
                        "type": "number",
                        "unit": "min",
                        "min": 0,
                    },
                    {
                        "key": "notes",
                        "label": {"en": "Notes", "uk": "Нотатки"},
                        "type": "longtext",
                    },
                ],
            },
        },
    ]


# ---------------------------------------------------------------------------
# Entry generators — 1..10/day for each tracker
# ---------------------------------------------------------------------------


def seed_entries(api: APIClient, trackers, days: int) -> None:
    feeding, sleep, diaper, activity = trackers

    today = dt.datetime.now(dt.timezone.utc).replace(microsecond=0, second=0)
    total = 0

    print(f"seeding {days} days …")
    for offset in range(days, 0, -1):
        day = today - dt.timedelta(days=offset)

        # Feeding: 5–9 entries/day, varied amount and type
        for _ in range(random.randint(5, 9)):
            data = {
                "occurred_at": iso(rand_time_in(day)),
                "amount": random.choice([60, 90, 120, 150, 180, 210]),
                "type": random.choices(
                    ["bottle", "breast", "solid"], weights=[5, 4, 1]
                )[0],
            }
            if random.random() < 0.25:
                data["note"] = random.choice(
                    ["fussy", "fast", "slow start", "good appetite"]
                )
            api.create_entry(feeding["id"], data)
            total += 1

        # Sleep: 3–6/day. Naps 30–120 min; nighttime sleeps 6–10 h.
        for _ in range(random.randint(3, 6)):
            ts = rand_time_in(day)
            if ts.hour <= 4 or ts.hour >= 21:
                length_ms = random.randint(360, 600) * 60 * 1000
            else:
                length_ms = random.randint(30, 120) * 60 * 1000
            data = {
                "occurred_at": iso(ts),
                "length": length_ms,
                "where": random.choice(["crib", "stroller", "parents_bed", "car"]),
                "woke_up": random.random() < 0.3,
            }
            api.create_entry(sleep["id"], data)
            total += 1

        # Diaper: 4–10/day
        for _ in range(random.randint(4, 10)):
            data = {
                "occurred_at": iso(rand_time_in(day)),
                "type": random.choices(
                    ["wet", "dirty", "both", "dry"], weights=[6, 3, 2, 1]
                )[0],
                "rash": random.random() < 0.1,
            }
            api.create_entry(diaper["id"], data)
            total += 1

        # Activity: 1–4/day, daytime only
        for _ in range(random.randint(1, 4)):
            ts = rand_time_in(day, hour_min=7, hour_max=20)
            kinds = random.sample(
                ["tummy_time", "walk", "bath", "play", "reading"],
                k=random.randint(1, 3),
            )
            data = {
                "occurred_at": iso(ts),
                "kinds": kinds,
                "duration_min": random.choice([10, 15, 20, 30, 45, 60]),
            }
            if random.random() < 0.2:
                data["notes"] = "great mood today"
            api.create_entry(activity["id"], data)
            total += 1

        if offset % 10 == 0:
            print(f"  …through day -{offset:>3}: {total} entries so far")

    print(f"\ndone. created {total} entries across {len(trackers)} trackers.")


def rand_time_in(day: dt.datetime, hour_min: int = 0, hour_max: int = 23) -> dt.datetime:
    return day.replace(
        hour=random.randint(hour_min, hour_max),
        minute=random.randint(0, 59),
        second=random.randint(0, 59),
    )


def iso(t: dt.datetime) -> str:
    return t.astimezone(dt.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


if __name__ == "__main__":
    main()
