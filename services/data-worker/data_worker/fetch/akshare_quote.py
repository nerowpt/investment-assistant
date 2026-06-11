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


_INDEX_NAMES = {
    "000300": "沪深300",
    "000905": "中证500",
    "399006": "创业板指",
    "000016": "上证50",
}


def fetch_market_snapshot(index_codes: list[str]) -> dict[str, Any]:
    """拉取指数当日行情（研究档案·市场基准包）。"""
    import akshare as ak

    indices: list[dict[str, Any]] = []
    lines: list[str] = []
    for code in index_codes:
        code = str(code).strip()
        name = _INDEX_NAMES.get(code, code)
        close = 0.0
        change_pct = 0.0
        try:
            df = _call_akshare("index_zh_a_hist", ak.index_zh_a_hist, symbol=code, period="daily")
            if df is not None and not df.empty:
                last = df.iloc[-1]
                close = float(last.get("收盘", 0) or 0)
                if len(df) >= 2:
                    prev = float(df.iloc[-2].get("收盘", 0) or 0)
                    if prev > 0:
                        change_pct = (close - prev) / prev * 100
        except Exception:
            pass
        indices.append({"code": code, "name": name, "close": close, "change_pct": change_pct})
        if close > 0:
            lines.append(f"{name}({code}) 收盘 {close:.2f}，涨跌 {change_pct:.2f}%")
    summary = "；".join(lines) if lines else "指数数据暂不可用"
    return {
        "provenance": {"source": "akshare", "tier": "B", "captured_at": _now_iso()},
        "indices": indices,
        "summary": summary,
    }


def _fetch_eastmoney_stock_meta(code: str) -> dict[str, str]:
    """东财 push2 个股元数据：简称、行业（f127）、地区（f128）。"""
    market_code = 1 if code.startswith("6") else 0
    params = {
        "ut": "bd1d9ddb04089700cf9c27f6f7426281",
        "fltt": "2",
        "invt": "2",
        "fields": "f57,f58,f127,f128",
        "secid": f"{market_code}.{code}",
    }
    data = _em_push2_get("/api/qt/stock/get", params, timeout=15).get("data") or {}
    industry = str(data.get("f127") or "").strip()
    if not industry:
        industry = _fetch_eastmoney_f10_industry(code)
    return {
        "name": str(data.get("f58") or "").strip(),
        "industry": industry,
        "sector": str(data.get("f128") or "").strip(),
    }


def _request_with_retry(label: str, fn, retries: int = 2):
    """东财 HTTP 抖动时重试。"""
    import time

    last_err: Exception | None = None
    for attempt in range(retries):
        try:
            return fn()
        except Exception as exc:  # noqa: BLE001
            last_err = exc
            if attempt < retries - 1:
                time.sleep(0.6)
    raise RuntimeError(f"{label} 失败: {last_err}") from last_err


def _fetch_eastmoney_f10_profile(code: str) -> dict[str, str]:
    """F10 公司概况：简称、申万/证监会行业（jbzl 为 list）。"""
    import requests

    prefix = "SH" if code.startswith("6") else "SZ"
    url = "https://emweb.securities.eastmoney.com/PC_HSF10/CompanySurvey/PageAjax"
    params = {"code": f"{prefix}{code}"}
    headers = {
        "User-Agent": (
            "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "
            "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
        ),
        "Referer": (
            f"https://emweb.securities.eastmoney.com/PC_HSF10/CompanySurvey/"
            f"CompanySurveyIndex?type=web&code={prefix}{code}"
        ),
    }
    out = {"name": "", "industry": "", "em2016": "", "csrc": ""}
    try:
        r = requests.get(url, params=params, headers=headers, timeout=12)
        r.raise_for_status()
        jbzl = r.json().get("jbzl")
        row: dict[str, Any] = {}
        if isinstance(jbzl, list) and jbzl:
            row = jbzl[0] if isinstance(jbzl[0], dict) else {}
        elif isinstance(jbzl, dict):
            row = jbzl
        out["name"] = str(row.get("SECURITY_NAME_ABBR") or "").strip()
        out["em2016"] = str(row.get("EM2016") or "").strip()
        out["csrc"] = str(row.get("INDUSTRYCSRC1") or "").strip()
        for raw in (out["em2016"], out["csrc"]):
            if raw:
                parts = [p.strip() for p in raw.replace("—", "-").split("-") if p.strip()]
                if parts:
                    out["industry"] = parts[-1]
                    break
    except Exception:
        pass
    return out


def _fetch_eastmoney_f10_industry(code: str) -> str:
    return _fetch_eastmoney_f10_profile(code).get("industry", "")


def _fetch_f10_industry_pe_stats(code: str) -> dict[str, float]:
    """F10 行业分析：行业平均 PE、近3年行业均值峰值（单次 HTTP）。"""
    import requests

    prefix = "SH" if code.startswith("6") else "SZ"
    url = "https://emweb.securities.eastmoney.com/PC_HSF10/IndustryAnalysis/PageAjax"
    params = {"code": f"{prefix}{code}"}
    headers = {
        "User-Agent": (
            "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "
            "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
        ),
        "Referer": (
            f"https://emweb.securities.eastmoney.com/PC_HSF10/IndustryAnalysis/"
            f"Index?type=web&code={prefix}{code}"
        ),
    }
    out = {"avg_pe_ttm": 0.0, "hist_max_pe": 0.0}
    try:
        r = requests.get(url, params=params, headers=headers, timeout=10)
        r.raise_for_status()
        gzbj = r.json().get("gzbj") or []
        for row in gzbj:
            if not isinstance(row, dict):
                continue
            label = str(row.get("CORRE_SECURITY_NAME") or row.get("SECURITY_CODE") or "")
            if label != "行业平均":
                continue
            pe_vals: list[float] = []
            for key in ("PE_TTM", "PE", "PE_1Y", "PE_2Y", "PE_3Y"):
                raw = row.get(key)
                if raw is None:
                    continue
                try:
                    val = float(raw)
                except (TypeError, ValueError):
                    continue
                if val > 0:
                    pe_vals.append(val)
            if pe_vals:
                out["avg_pe_ttm"] = float(row.get("PE_TTM") or pe_vals[0])
                out["hist_max_pe"] = max(pe_vals)
            break
    except Exception:
        pass
    return out


# 常见申万行业 → 东财板块 QuoteID（search/clist 不可达时的兜底）
_KNOWN_INDUSTRY_BOARDS: dict[str, tuple[str, str]] = {
    "证券": ("90.BK0473", "证券Ⅱ"),
    "券商": ("90.BK0473", "证券Ⅱ"),
    "银行": ("90.BK0475", "银行Ⅱ"),
    "保险": ("90.BK0474", "保险Ⅱ"),
    "白酒": ("90.BK0896", "白酒Ⅱ"),
    "半导体": ("90.BK1036", "半导体"),
    "光伏": ("90.BK1031", "光伏设备"),
    "电池": ("90.BK1033", "电池"),
}


def _expand_industry_hints(*values: str) -> list[str]:
    """展开行业/板块别名为搜索关键词。"""
    hints: list[str] = []
    for raw in values:
        if not raw or raw.strip() in ("", "未知"):
            continue
        text = raw.strip()
        hints.append(text)
        for sep in ("-", "/", "—"):
            if sep in text:
                hints.extend(p.strip() for p in text.split(sep) if p.strip())
        base = text.rstrip("ⅢⅡⅠ")
        if base and base != text:
            hints.append(base)
    if any("证券" in h for h in hints):
        hints.extend(["证券", "券商", "证券Ⅱ"])
    deduped = list(dict.fromkeys(hints))
    return [h for h in deduped if h and h != "未知"]


def _search_eastmoney_board(hint: str) -> dict[str, str]:
    """东财搜索 API：行业名 → 板块 QuoteID。"""
    import requests

    url = "https://searchapi.eastmoney.com/api/suggest/get"
    params = {
        "input": hint,
        "type": "14",
        "token": "D43BF725234BC936D6FB241F126B75D8",
        "count": 10,
    }
    headers = {
        **_eastmoney_clist_headers(),
        "Referer": "https://so.eastmoney.com/",
    }

    def _do() -> dict[str, str]:
        r = requests.get(url, params=params, headers=headers, timeout=12)
        r.raise_for_status()
        items = (r.json().get("QuotationCodeTable") or {}).get("Data") or []
        best: dict[str, str] = {"name": "", "quote_id": "", "code": ""}
        best_score = -1
        for it in items:
            if str(it.get("Classify") or "") != "BK":
                continue
            name = str(it.get("Name") or "").strip()
            score = 0
            if name == hint:
                score = 100
            elif hint in name or name in hint:
                score = 80
            elif hint.rstrip("ⅢⅡⅠ") in name:
                score = 60
            if score > best_score:
                best_score = score
                best = {
                    "name": name,
                    "quote_id": str(it.get("QuoteID") or "").strip(),
                    "code": str(it.get("Code") or "").strip(),
                }
        return best

    try:
        return _request_with_retry("board_search", _do)
    except Exception:
        return {"name": "", "quote_id": "", "code": ""}


def _fetch_board_pe_pb_by_quote_id(quote_id: str) -> dict[str, Any]:
    """板块指数 push2 查询 PE/PB（f162/f167）。"""
    if not quote_id:
        return {"name": "", "pe": 0.0, "pb": 0.0}
    url = "https://push2.eastmoney.com/api/qt/stock/get"
    params = {
        "ut": "bd1d9ddb04089700cf9c27f6f7426281",
        "fltt": "2",
        "invt": "2",
        "secid": quote_id,
        "fields": "f14,f162,f167",
    }

    def _do() -> dict[str, Any]:
        data = _em_push2_get("/api/qt/stock/get", params).get("data") or {}
        pe = float(data.get("f162") or 0)
        pb_raw = data.get("f167")
        pb = 0.0
        if pb_raw not in (None, "", "-"):
            try:
                pb = float(pb_raw)
            except (TypeError, ValueError):
                pb = 0.0
        return {
            "name": str(data.get("f14") or "").strip(),
            "pe": pe,
            "pb": pb,
        }

    try:
        return _request_with_retry("board_quote", _do)
    except Exception:
        return {"name": "", "pe": 0.0, "pb": 0.0}


def fetch_stock_basic(code: str) -> dict[str, str]:
    """个股简称、行业；多源 fallback，失败不抛异常。"""
    code = code.strip()
    name = ""
    industry = ""
    sector = ""

    try:
        meta = _fetch_eastmoney_stock_meta(code)
        name = meta.get("name", "")
        industry = meta.get("industry", "")
        sector = meta.get("sector", "")
    except Exception:
        pass

    if not name:
        try:
            em = _fetch_eastmoney_quote(code)
            if em:
                name = str(em.get("name") or "").strip()
        except Exception:
            pass

    if not name:
        try:
            sina = _fetch_sina_quote(code)
            name = str(sina.get("name") or "").strip()
        except Exception:
            pass

    if not industry or not name:
        try:
            import akshare as ak

            info = _row_map(
                _call_akshare("stock_individual_info_em", ak.stock_individual_info_em, symbol=code)
            )
            if not name:
                name = str(info.get("股票简称") or info.get("证券简称") or info.get("名称") or "").strip()
            if not industry:
                industry = str(info.get("行业") or info.get("所属行业") or "").strip()
        except Exception:
            pass

    if not industry and sector:
        industry = sector
    return {"name": name or code, "industry": industry or "未知", "sector": sector}


def _fetch_eastmoney_daily_kline(code: str, limit: int = 120) -> list[tuple[str, float]]:
    """东财日 K 成交量序列（手），用于量能分析。"""
    import requests
    import time

    market = 1 if code.startswith("6") else 0
    url = "https://push2his.eastmoney.com/api/qt/stock/kline/get"
    params = {
        "secid": f"{market}.{code}",
        "ut": "bd1d9ddb04089700cf9c27f6f7426281",
        "fields1": "f1,f2,f3,f4,f5,f6",
        "fields2": "f51,f52,f53,f54,f55,f56,f57,f58",
        "klt": "101",
        "fqt": "1",
        "beg": "0",
        "end": "20500101",
        "lmt": str(limit),
        "_": str(int(time.time() * 1000)),
    }
    r = requests.get(url, params=params, headers=_eastmoney_clist_headers(), timeout=15)
    r.raise_for_status()
    klines = (r.json().get("data") or {}).get("klines") or []
    out: list[tuple[str, float]] = []
    for line in klines:
        parts = line.split(",")
        if len(parts) < 6:
            continue
        try:
            out.append((parts[0][:10], float(parts[5])))
        except ValueError:
            continue
    return out


def _fetch_tencent_daily_kline(code: str, limit: int = 120) -> list[tuple[str, float]]:
    """腾讯日 K fallback（手）。"""
    import requests

    prefix = "sh" if code.startswith("6") else "sz"
    url = "https://web.ifzq.gtimg.cn/appstock/app/fqkline/get"
    params = {"param": f"{prefix}{code},day,,,{limit},qfq"}
    headers = {"Referer": "https://gu.qq.com/"}
    r = requests.get(url, params=params, headers=headers, timeout=15)
    r.raise_for_status()
    payload = r.json()
    node = (((payload.get("data") or {}).get(prefix + code) or {}).get("qfqday")) or []
    out: list[tuple[str, float]] = []
    for row in node:
        if not isinstance(row, (list, tuple)) or len(row) < 6:
            continue
        try:
            out.append((str(row[0])[:10], float(row[5])))
        except (TypeError, ValueError):
            continue
    return out


def _fetch_daily_volume_series(code: str, limit: int = 120) -> tuple[list[tuple[str, float]], str]:
    """日成交量序列 + 来源标识；东财 → 腾讯 → akshare。"""
    for fn, source in (
        (_fetch_eastmoney_daily_kline, "eastmoney"),
        (_fetch_tencent_daily_kline, "tencent"),
    ):
        try:
            rows = fn(code, limit)
            if rows:
                return rows, source
        except Exception:
            pass
    try:
        import akshare as ak

        prefix = _market_prefix(code)
        hist = _call_akshare(
            "stock_zh_a_hist",
            ak.stock_zh_a_hist,
            symbol=f"{prefix}{code}",
            period="daily",
            adjust="qfq",
        )
        if hist is not None and not hist.empty and "成交量" in hist.columns:
            rows = [
                (str(hist.iloc[i].get("日期", ""))[:10], float(hist.iloc[i]["成交量"]))
                for i in range(len(hist))
            ]
            if rows:
                return rows[-limit:], "akshare"
    except Exception:
        pass
    return [], "unknown"


def _eastmoney_clist_headers() -> dict[str, str]:
    return {
        "User-Agent": (
            "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "
            "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
        ),
        "Referer": "https://quote.eastmoney.com/",
    }


_PUSH2_HOSTS = (
    "https://push2.eastmoney.com",
    "https://82.push2.eastmoney.com",
    "https://push2delay.eastmoney.com",
)


def _em_push2_get(path: str, params: dict[str, str], timeout: int = 12) -> dict[str, Any]:
    """东财 push2 多节点重试。"""
    import requests

    last_err: Exception | None = None
    for host in _PUSH2_HOSTS:
        try:
            r = requests.get(
                f"{host}{path}",
                params=params,
                headers=_eastmoney_clist_headers(),
                timeout=timeout,
            )
            r.raise_for_status()
            return r.json()
        except Exception as exc:  # noqa: BLE001
            last_err = exc
    if last_err:
        raise last_err
    return {}


def _board_from_quote_id(quote_id: str, fallback_name: str = "") -> dict[str, Any]:
    metrics = _fetch_board_pe_pb_by_quote_id(quote_id)
    if metrics.get("pe", 0) > 0 or metrics.get("pb", 0) > 0:
        return {
            "name": metrics.get("name") or fallback_name,
            "pe": metrics.get("pe", 0),
            "pb": metrics.get("pb", 0),
        }
    return {"name": "", "pe": 0.0, "pb": 0.0}


def _board_from_known_industry(hint: str) -> dict[str, Any]:
    key = hint.rstrip("ⅢⅡⅠ")
    mapped = _KNOWN_INDUSTRY_BOARDS.get(hint) or _KNOWN_INDUSTRY_BOARDS.get(key)
    if not mapped:
        return {"name": "", "pe": 0.0, "pb": 0.0}
    quote_id, board_name = mapped
    return _board_from_quote_id(quote_id, board_name)


def _fetch_eastmoney_industry_board_from_clist(hints: list[str]) -> dict[str, Any]:
    """东财行业板块 clist 匹配（网络不稳，作首选但可失败）。"""
    import requests

    if not hints:
        return {"name": "", "pe": 0.0, "pb": 0.0}
    url = "https://push2.eastmoney.com/api/qt/clist/get"
    params = {
        "pn": "1",
        "pz": "200",
        "po": "1",
        "np": "1",
        "ut": "bd1d9ddb04089700cf9c27f6f7426281",
        "fltt": "2",
        "invt": "2",
        "fid": "f3",
        "fs": "m:90+t:2",
        "fields": "f12,f14,f9,f23",
    }

    def _do() -> dict[str, Any]:
        r = requests.get(url, params=params, headers=_eastmoney_clist_headers(), timeout=12)
        r.raise_for_status()
        rows = (r.json().get("data") or {}).get("diff") or []
        best_name = ""
        best_pe = best_pb = 0.0
        best_score = -1
        for row in rows:
            board = str(row.get("f14") or "").strip()
            if not board:
                continue
            score = 0
            for h in hints:
                if h == board:
                    score = 100
                elif h in board or board in h:
                    score = max(score, 60)
                elif any(part and (part in board or board in part) for part in h.replace("/", " ").split()):
                    score = max(score, 40)
            if score <= 0:
                continue
            pe = float(row.get("f9") or 0)
            pb = float(row.get("f23") or 0)
            if score > best_score or (score == best_score and pe > best_pe):
                best_score = score
                best_name = board
                best_pe, best_pb = pe, pb
        return {"name": best_name, "pe": best_pe, "pb": best_pb}

    try:
        return _request_with_retry("industry_clist", _do)
    except Exception:
        return {"name": "", "pe": 0.0, "pb": 0.0}


def _fetch_eastmoney_industry_board(hints: list[str]) -> dict[str, Any]:
    """行业板块 PE/PB：搜索 → 已知映射 → clist。"""
    hints = _expand_industry_hints(*hints)
    if not hints:
        return {"name": "", "pe": 0.0, "pb": 0.0}

    for hint in sorted(hints, key=len, reverse=True):
        found = _search_eastmoney_board(hint)
        quote_id = found.get("quote_id") or ""
        if not quote_id and found.get("code"):
            quote_id = f"90.{found['code']}"
        if quote_id:
            board = _board_from_quote_id(quote_id, found.get("name") or hint)
            if board.get("pe", 0) > 0 or board.get("pb", 0) > 0:
                return board
        board = _board_from_known_industry(hint)
        if board.get("pe", 0) > 0 or board.get("pb", 0) > 0:
            return board

    board = _fetch_eastmoney_industry_board_from_clist(hints)
    if board.get("pe", 0) > 0 or board.get("pb", 0) > 0:
        return board
    return {"name": "", "pe": 0.0, "pb": 0.0}


def fetch_sector_valuation(code: str) -> dict[str, Any]:
    """板块/行业估值事实包（PE 等为行业板块指标，非投资建议）。"""
    code = code.strip()
    name = code
    industry = "未知"
    sector_hint = ""
    f10 = _fetch_eastmoney_f10_profile(code)
    try:
        meta = _fetch_eastmoney_stock_meta(code)
        name = meta.get("name") or f10.get("name") or code
        industry = meta.get("industry") or f10.get("industry") or "未知"
        sector_hint = meta.get("sector") or ""
    except Exception:
        name = f10.get("name") or code
        industry = f10.get("industry") or "未知"
        sector_hint = ""
        try:
            em = _fetch_eastmoney_quote(code)
            if em:
                name = str(em.get("name") or name)
        except Exception:
            pass

    # f128 为地区（如「北京板块」），不可用于行业板块 PE 匹配
    hints = _expand_industry_hints(industry, f10.get("em2016", ""), f10.get("csrc", ""))
    board = _fetch_eastmoney_industry_board(hints)
    industry_name = board.get("name") or industry
    industry_pe = float(board.get("pe") or 0)
    industry_pb = float(board.get("pb") or 0)
    pe_stats = _fetch_f10_industry_pe_stats(code)
    avg_pe = float(pe_stats.get("avg_pe_ttm") or 0)
    hist_max_pe = float(pe_stats.get("hist_max_pe") or 0)

    lines = [f"标的：{name} ({code})", f"所属行业：{industry}"]
    if industry_pe > 0 or industry_pb > 0:
        lines.append(f"行业板块：{industry_name}")
        if industry_pe > 0:
            lines.append(f"板块当前市盈率：{industry_pe:.2f}")
        if industry_pb > 0:
            lines.append(f"板块市净率：{industry_pb:.2f}")
    else:
        lines.append("行业板块估值：暂未能拉取到板块 PE/PB（可稍后重试）")
    if avg_pe > 0:
        lines.append(f"行业平均市盈率（TTM）：{avg_pe:.2f}")
    if hist_max_pe > 0:
        lines.append(f"历史最高市盈率（行业均值口径，近3年统计峰值）：{hist_max_pe:.2f}")
    if industry_pe > 0 and avg_pe > 0:
        lines.append(f"当前板块 PE / 行业平均 PE：{industry_pe:.2f} / {avg_pe:.2f}")
    lines.append("说明：以上为行业板块指标快照，不构成估值高低判断。")
    body = "\n".join(lines)
    summary = industry_name
    if industry_pe > 0:
        summary += f" PE≈{industry_pe:.1f}"
    if avg_pe > 0:
        summary += f"；行业均{avg_pe:.1f}"
    return {
        "provenance": {"source": "eastmoney", "tier": "B", "captured_at": _now_iso()},
        "code": code,
        "stock_name": name,
        "title": f"{name} 板块估值 {datetime.now().strftime('%Y-%m-%d')}",
        "summary": summary,
        "body": body,
    }


def fetch_volume_analysis(code: str) -> dict[str, Any]:
    """成交量对比：近日 vs 历史均量，标注缩量/正常/放量高峰。"""
    code = code.strip()
    name = code
    try:
        meta = _fetch_eastmoney_stock_meta(code)
        name = meta.get("name") or code
    except Exception:
        try:
            em = _fetch_eastmoney_quote(code)
            if em:
                name = str(em.get("name") or code)
        except Exception:
            pass

    rows, source = _fetch_daily_volume_series(code)
    if not rows:
        raise LookupError(f"无成交量历史: {code}（东财/腾讯/akshare 均不可达）")

    vols = [v for _, v in rows]
    last_date, last_vol = rows[-1][0], vols[-1]
    tail = vols[-60:] if len(vols) >= 60 else vols
    avg5 = sum(vols[-5:]) / min(len(vols), 5)
    avg20 = sum(vols[-20:]) / min(len(vols), 20)
    avg60 = sum(tail) / len(tail)
    window = tail
    pct_rank = float(sum(1 for v in window if v < last_vol)) / max(len(window), 1) * 100
    vs60 = (last_vol / avg60 - 1) * 100 if avg60 > 0 else 0.0
    if vs60 < -25 or pct_rank < 20:
        level = "缩量偏低"
    elif vs60 > 60 or pct_rank > 85:
        level = "放量高峰"
    elif vs60 > 25 or pct_rank > 65:
        level = "温和放量"
    else:
        level = "接近历史常态"
    lines = [
        f"标的：{name} ({code})",
        f"最近交易日：{last_date}",
        f"当日成交量：{last_vol:,.0f} 手",
        f"近 5 日均量：{avg5:,.0f} 手",
        f"近 20 日均量：{avg20:,.0f} 手",
        f"近 60 日均量：{avg60:,.0f} 手",
        f"当日 vs 60 日均量：{vs60:+.1f}%",
        f"近 60 日分位（低于当日占比）：{pct_rank:.0f}%",
        f"量能状态（规则标签）：{level}",
        "说明：标签仅基于成交量统计对比，不代表买卖建议。",
    ]
    body = "\n".join(lines)
    summary = f"{level}；当日 {last_vol/1e4:.1f}万手，60日均 {avg60/1e4:.1f}万手"
    return {
        "provenance": {"source": source, "tier": "A", "captured_at": _now_iso()},
        "code": code,
        "stock_name": name,
        "title": f"{name} 成交量分析 {last_date}",
        "summary": summary,
        "body": body,
    }
