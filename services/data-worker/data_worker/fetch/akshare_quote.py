"""akshare 行情拉取。"""

from __future__ import annotations

from datetime import datetime, timezone
from typing import Any


def _now_iso() -> str:
    return datetime.now(timezone.utc).astimezone().isoformat()


def _market_prefix(code: str) -> str:
    """A 股代码 → 东财 market 前缀。"""
    if code.startswith(("5", "6", "9")):
        return "sh"
    return "sz"


def _row_map(df) -> dict[str, str]:
    """将 item/value 两列 DataFrame 转为 dict。"""
    if df is None or df.empty:
        return {}
    cols = list(df.columns)
    if len(cols) < 2:
        return {}
    key_col, val_col = cols[0], cols[1]
    return {str(r[key_col]): r[val_col] for _, r in df.iterrows()}


def _call_akshare(label: str, fn, *args, **kwargs):
    """调用 akshare 接口，网络抖动时重试一次。"""
    import time

    last_err: Exception | None = None
    for attempt in range(2):
        try:
            return fn(*args, **kwargs)
        except Exception as exc:  # noqa: BLE001
            last_err = exc
            if attempt == 0:
                time.sleep(0.8)
    raise RuntimeError(f"{label} 失败: {last_err}") from last_err


def _fetch_eastmoney_quote(code: str) -> dict[str, Any] | None:
    """东财 push2 单标的行情（带浏览器头，绕过裸 requests 被拒）。"""
    import requests

    market_code = 1 if code.startswith("6") else 0
    url = "https://push2.eastmoney.com/api/qt/stock/get"
    params = {
        "fltt": "2",
        "invt": "2",
        "fields": "f43,f57,f58,f169,f170",
        "secid": f"{market_code}.{code}",
    }
    headers = {
        "User-Agent": (
            "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "
            "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
        ),
        "Referer": "https://quote.eastmoney.com/",
    }
    r = requests.get(url, params=params, headers=headers, timeout=15)
    r.raise_for_status()
    data = r.json().get("data") or {}
    price = float(data.get("f43") or 0)
    if price <= 0:
        return None
    return {
        "name": str(data.get("f58") or ""),
        "price": price,
        "change_amount": float(data.get("f169") or 0),
        "change_pct": float(data.get("f170") or 0),
    }


def _fetch_sina_quote(code: str) -> dict[str, Any]:
    """新浪 hq.sinajs.cn 实时行情（国内网络通常更稳定）。"""
    import requests

    prefix = "sh" if code.startswith("6") else "sz"
    url = f"https://hq.sinajs.cn/list={prefix}{code}"
    headers = {"Referer": "https://finance.sina.com.cn/"}
    r = requests.get(url, headers=headers, timeout=15)
    r.raise_for_status()
    text = r.content.decode("gbk", errors="replace")
    # var hq_str_sh600519="名称,今开,昨收,现价,..."
    if "=" not in text or '"' not in text:
        raise LookupError(f"新浪返回异常: {text[:80]}")
    payload = text.split('"', 2)[1]
    parts = payload.split(",")
    if len(parts) < 4:
        raise LookupError(f"新浪字段不足: {payload[:80]}")
    name = parts[0]
    yesterday = float(parts[2] or 0)
    price = float(parts[3] or 0)
    if price <= 0:
        raise LookupError(f"新浪无有效现价: {code}")
    change_amount = price - yesterday if yesterday > 0 else 0.0
    change_pct = (change_amount / yesterday * 100) if yesterday > 0 else 0.0
    return {
        "name": name,
        "price": price,
        "change_amount": change_amount,
        "change_pct": change_pct,
    }


def fetch_quote(code: str) -> dict[str, Any]:
    """拉取 A 股实时行情（单标的 API，避免全市场 spot 表）。"""
    import akshare as ak

    code = code.strip()
    if len(code) != 6 or not code.isdigit():
        raise ValueError(f"非法股票代码: {code}")

    name = ""
    price = 0.0
    change_pct = 0.0
    change_amount = 0.0
    source = "akshare"

    # 1) 东财 push2（直连，带 Referer）
    try:
        em = _fetch_eastmoney_quote(code)
        if em:
            name, price = em["name"], em["price"]
            change_pct, change_amount = em["change_pct"], em["change_amount"]
            source = "eastmoney"
    except Exception:
        pass

    # 2) 新浪 fallback（health 通过后 quote 仍失败时常见）
    if price <= 0:
        try:
            sina = _fetch_sina_quote(code)
            name = sina["name"]
            price = sina["price"]
            change_pct = sina["change_pct"]
            change_amount = sina["change_amount"]
            source = "sina"
        except Exception:
            pass

    # 3) akshare 封装接口（部分环境可用）
    if price <= 0:
        try:
            bid = _row_map(_call_akshare("stock_bid_ask_em", ak.stock_bid_ask_em, symbol=code))
            if bid:
                name = str(bid.get("名称", "") or bid.get("股票名称", ""))
                price = float(bid.get("最新", bid.get("最新价", 0)) or 0)
                change_pct = float(bid.get("涨幅", bid.get("涨跌幅", 0)) or 0)
                change_amount = float(bid.get("涨跌", bid.get("涨跌额", 0)) or 0)
        except Exception:
            pass

    # 4) 历史收盘价兜底
    if price <= 0:
        try:
            prefix = _market_prefix(code)
            hist = _call_akshare(
                "stock_zh_a_hist",
                ak.stock_zh_a_hist,
                symbol=f"{prefix}{code}",
                period="daily",
                adjust="qfq",
            )
            if hist is not None and not hist.empty:
                last = hist.iloc[-1]
                price = float(last.get("收盘", 0) or 0)
                if change_pct == 0 and len(hist) >= 2:
                    prev = float(hist.iloc[-2].get("收盘", 0) or 0)
                    if prev > 0:
                        change_pct = (price - prev) / prev * 100
                        change_amount = price - prev
                source = "akshare_hist"
        except Exception:
            pass

    if price <= 0:
        raise LookupError(
            f"未找到标的行情: {code}（东财/新浪均不可达，请检查网络或代理；非 H3 安装问题）"
        )

    return {
        "provenance": {
            "source": source,
            "tier": "A",
            "captured_at": _now_iso(),
            "url": "https://akshare.akfamily.xyz",
        },
        "code": code,
        "name": name or code,
        "price": price,
        "change_pct": change_pct,
        "change_amount": change_amount,
        "trade_date": datetime.now().strftime("%Y-%m-%d"),
    }


def _fetch_eastmoney_valuation(code: str) -> dict[str, Any] | None:
    """东财 push2 估值字段：f162=PE(动) f167=PB f173=市销率 f92=股息率等。"""
    import requests

    market_code = 1 if code.startswith("6") else 0
    url = "https://push2.eastmoney.com/api/qt/stock/get"
    params = {
        "fltt": "2",
        "invt": "2",
        "fields": "f57,f162,f167,f173",
        "secid": f"{market_code}.{code}",
    }
    headers = {
        "User-Agent": (
            "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "
            "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
        ),
        "Referer": "https://quote.eastmoney.com/",
    }
    r = requests.get(url, params=params, headers=headers, timeout=15)
    r.raise_for_status()
    data = r.json().get("data") or {}
    pe = float(data.get("f162") or 0)
    pb = float(data.get("f167") or 0)
    ps = float(data.get("f173") or 0)
    if pe <= 0 and pb <= 0 and ps <= 0:
        return None
    return {"pe_ttm": pe, "pb": pb, "ps_ttm": ps, "source": "eastmoney"}


def _fetch_datacenter_valuation(code: str) -> dict[str, Any] | None:
    """东财 datacenter 估值分析表（fallback）。"""
    import requests

    url = "https://datacenter-web.eastmoney.com/api/data/v1/get"
    params = {
        "sortColumns": "TRADE_DATE",
        "sortTypes": "-1",
        "pageSize": "1",
        "pageNumber": "1",
        "reportName": "RPT_VALUEANALYSIS_DET",
        "columns": "ALL",
        "source": "WEB",
        "client": "WEB",
        "filter": f'(SECURITY_CODE="{code}")',
    }
    headers = {
        "User-Agent": "Mozilla/5.0",
        "Referer": "https://data.eastmoney.com/",
    }
    r = requests.get(url, params=params, headers=headers, timeout=15)
    r.raise_for_status()
    rows = (r.json().get("result") or {}).get("data") or []
    if not rows:
        return None
    row = rows[0]
    return {
        "pe_ttm": float(row.get("PE_TTM") or 0),
        "pb": float(row.get("PB_MRQ") or 0),
        "ps_ttm": float(row.get("PS_TTM") or 0),
        "as_of_date": str(row.get("TRADE_DATE", ""))[:10],
        "source": "eastmoney_datacenter",
    }


def fetch_valuation(code: str) -> dict[str, Any]:
    """拉取估值指标（PE/PB/PS；分位 MVP-1 暂为 0）。"""
    code = code.strip()
    if len(code) != 6 or not code.isdigit():
        raise ValueError(f"非法股票代码: {code}")

    pe_ttm = pb = ps_ttm = 0.0
    as_of = datetime.now().strftime("%Y-%m-%d")
    source = "eastmoney"

    # datacenter 在国内网络通常比 push2 更稳，优先尝试
    try:
        dc = _fetch_datacenter_valuation(code)
        if dc:
            pe_ttm = dc.get("pe_ttm", pe_ttm)
            pb = dc.get("pb", pb)
            ps_ttm = dc.get("ps_ttm", ps_ttm) or ps_ttm
            if dc.get("as_of_date"):
                as_of = dc["as_of_date"]
            source = dc.get("source", source)
    except Exception:
        pass

    if pe_ttm <= 0 and pb <= 0:
        try:
            em = _fetch_eastmoney_valuation(code)
            if em:
                pe_ttm, pb, ps_ttm = em["pe_ttm"], em["pb"], em["ps_ttm"]
                source = em["source"]
        except Exception:
            pass

    if pe_ttm <= 0 and pb <= 0:
        try:
            dc = _fetch_datacenter_valuation(code)
            if dc:
                pe_ttm = dc.get("pe_ttm", pe_ttm)
                pb = dc.get("pb", pb)
                ps_ttm = dc.get("ps_ttm", ps_ttm) or ps_ttm
                if dc.get("as_of_date"):
                    as_of = dc["as_of_date"]
                source = dc.get("source", source)
        except Exception:
            pass

    if pe_ttm <= 0 and pb <= 0:
        raise LookupError(
            f"未找到估值数据: {code}（东财不可达；非 H3 安装问题，请检查网络）"
        )

    return {
        "provenance": {
            "source": source,
            "tier": "A",
            "captured_at": _now_iso(),
        },
        "code": code,
        "pe_ttm": pe_ttm,
        "pb": pb,
        "ps_ttm": ps_ttm,
        "pe_percentile": 0.0,
        "pb_percentile": 0.0,
        "as_of_date": as_of,
    }


def fetch_announcements(codes: list[str], since: str = "") -> tuple[list[dict], list[dict]]:
    """拉取公告列表（MVP-1 简版：按标的逐条，部分失败写入 errors）。"""
    import akshare as ak

    items: list[dict] = []
    errors: list[dict] = []
    for code in codes:
        code = code.strip()
        try:
            df = ak.stock_news_em(symbol=code)
            if df is None or df.empty:
                continue
            for _, row in df.head(10).iterrows():
                items.append(
                    {
                        "provenance": {
                            "source": "akshare",
                            "tier": "B",
                            "captured_at": _now_iso(),
                        },
                        "code": code,
                        "name": "",
                        "title": str(row.get("新闻标题", "")),
                        "published_at": str(row.get("发布时间", "")),
                        "url": str(row.get("新闻链接", "")),
                        "content_type": "announcement",
                        "summary": str(row.get("新闻标题", ""))[:200],
                    }
                )
        except Exception as exc:  # noqa: BLE001
            errors.append({"code": code, "message": str(exc)})
    _ = since
    return items, errors
