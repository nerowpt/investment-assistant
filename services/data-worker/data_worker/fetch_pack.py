"""CLI：研究扩展包拉取（Go subprocess 调用，不依赖 gRPC stub 版本）。"""

from __future__ import annotations

import contextlib
import io
import json
import os
import sys

# akshare/tqdm 进度条会污染 stdout，导致 Go 侧 JSON 解析失败
os.environ.setdefault("TQDM_DISABLE", "1")
os.environ.setdefault("PYTHONIOENCODING", "utf-8")


def _ensure_utf8_stdio() -> None:
    """Windows 默认 GBK stdout 会导致 Go 解析后中文乱码。"""
    for name in ("stdout", "stderr"):
        stream = getattr(sys, name, None)
        if stream is None:
            continue
        if hasattr(stream, "reconfigure"):
            try:
                stream.reconfigure(encoding="utf-8", errors="replace")  # type: ignore[union-attr]
            except Exception:
                pass
        elif hasattr(stream, "buffer"):
            setattr(
                sys,
                name,
                io.TextIOWrapper(stream.buffer, encoding="utf-8", errors="replace"),
            )


_ensure_utf8_stdio()

from data_worker.fetch import akshare_quote


def _emit(payload: dict) -> None:
    """仅向 stdout 写入一行 JSON（供 Go subprocess 解析）。"""
    sys.stdout.write(json.dumps(payload, ensure_ascii=False) + "\n")
    sys.stdout.flush()


def main() -> None:
    if len(sys.argv) < 3:
        sys.stderr.write("usage: python -m data_worker.fetch_pack <pack> <code>\n")
        sys.exit(1)
    pack, code = sys.argv[1].strip(), sys.argv[2].strip()
    fns = {
        "sector_valuation": akshare_quote.fetch_sector_valuation,
        "volume": akshare_quote.fetch_volume_analysis,
    }
    fn = fns.get(pack)
    if fn is None:
        _emit({"error": f"unknown pack: {pack}"})
        sys.exit(1)
    try:
        # 拉取过程中的日志/进度条重定向，避免与 JSON 混排
        capture = io.StringIO()
        with contextlib.redirect_stdout(capture), contextlib.redirect_stderr(capture):
            data = fn(code)
        _emit(data)
    except Exception as exc:  # noqa: BLE001
        _emit({"error": str(exc)})
        sys.exit(1)


if __name__ == "__main__":
    main()
