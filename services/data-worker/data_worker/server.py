"""gRPC DataWorker 服务实现。"""

from __future__ import annotations

import logging
import os
import sys
from concurrent import futures
from pathlib import Path

import grpc

# 将 pb 生成目录加入 path，使 dataworker.v1 / common.v1 可导入
_PB_ROOT = Path(__file__).resolve().parent / "pb"
if str(_PB_ROOT) not in sys.path:
    sys.path.insert(0, str(_PB_ROOT))

from common.v1 import provenance_pb2  # noqa: E402
from dataworker.v1 import dataworker_pb2  # noqa: E402
from dataworker.v1 import dataworker_pb2_grpc  # noqa: E402

from data_worker import __version__
from data_worker.fetch import akshare_quote

logging.basicConfig(level=logging.INFO, format="%(asctime)s [data-worker] %(message)s")
logger = logging.getLogger(__name__)


def _prov(source: str, tier: str, url: str = "") -> provenance_pb2.Provenance:
    from datetime import datetime, timezone

    captured = datetime.now(timezone.utc).astimezone().isoformat()
    return provenance_pb2.Provenance(source=source, tier=tier, captured_at=captured, url=url)


class DataWorkerServicer(dataworker_pb2_grpc.DataWorkerServicer):
    """DataWorker gRPC 服务；禁止访问 DATA_ROOT / SQLite。"""

    def HealthCheck(self, request, context):  # noqa: ARG002
        import platform

        return dataworker_pb2.HealthCheckResponse(
            ok=True,
            version=__version__,
            python_version=platform.python_version(),
            providers=["akshare"],
        )

    def FetchQuote(self, request, context):
        try:
            data = akshare_quote.fetch_quote(request.code)
        except Exception as exc:  # noqa: BLE001
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(exc))
            return dataworker_pb2.FetchQuoteResponse()
        p = data["provenance"]
        return dataworker_pb2.FetchQuoteResponse(
            provenance=_prov(p["source"], p["tier"], p.get("url", "")),
            code=data["code"],
            name=data["name"],
            price=data["price"],
            change_pct=data["change_pct"],
            change_amount=data["change_amount"],
            trade_date=data["trade_date"],
        )

    def FetchValuation(self, request, context):
        try:
            data = akshare_quote.fetch_valuation(request.code)
        except Exception as exc:  # noqa: BLE001
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(exc))
            return dataworker_pb2.FetchValuationResponse()
        p = data["provenance"]
        return dataworker_pb2.FetchValuationResponse(
            provenance=_prov(p["source"], p["tier"]),
            code=data["code"],
            pe_ttm=data["pe_ttm"],
            pb=data["pb"],
            ps_ttm=data["ps_ttm"],
            pe_percentile=data["pe_percentile"],
            pb_percentile=data["pb_percentile"],
            as_of_date=data["as_of_date"],
        )

    def FetchAnnouncements(self, request, context):  # noqa: ARG002
        items_raw, errors_raw = akshare_quote.fetch_announcements(list(request.codes), request.since)
        items = []
        for it in items_raw:
            p = it["provenance"]
            items.append(
                dataworker_pb2.AnnouncementItem(
                    provenance=_prov(p["source"], p["tier"]),
                    code=it["code"],
                    name=it.get("name", ""),
                    title=it["title"],
                    published_at=it["published_at"],
                    url=it.get("url", ""),
                    content_type=it.get("content_type", "announcement"),
                    summary=it.get("summary", ""),
                )
            )
        errors = [
            dataworker_pb2.FetchError(code=e["code"], message=e["message"]) for e in errors_raw
        ]
        return dataworker_pb2.FetchAnnouncementsResponse(items=items, errors=errors)

    def FetchMarketSnapshot(self, request, context):  # noqa: ARG002
        """MVP-1 简版：返回请求的指数占位（H5 snapshot 可扩展）。"""
        indices = []
        for code in request.index_codes:
            indices.append(
                dataworker_pb2.IndexSnapshot(code=code, name=code, close=0.0, change_pct=0.0)
            )
        return dataworker_pb2.FetchMarketSnapshotResponse(
            provenance=_prov("akshare", "B"),
            indices=indices,
            summary="market snapshot stub",
        )

    def ExtractDocument(self, request, context):  # noqa: ARG002
        """URL/文件/文本抽取（MVP-1 简版）。"""
        title = request.suggested_title or "未命名"
        text = ""
        if request.HasField("text"):
            text = request.text
        elif request.HasField("url"):
            text = f"URL 抽取预留: {request.url}"
        elif request.HasField("file_bytes"):
            text = request.file_bytes.decode("utf-8", errors="replace")[:8000]
        truncated = len(text) > 8000
        if truncated:
            text = text[:8000]
        return dataworker_pb2.ExtractDocumentResponse(
            provenance=_prov("manual", "B"),
            title=title,
            text=text,
            truncated=truncated,
            char_count=len(text),
        )


def serve(listen_addr: str, port_file: str | None = None) -> None:
    """启动 gRPC 服务并阻塞。"""
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=4))
    dataworker_pb2_grpc.add_DataWorkerServicer_to_server(DataWorkerServicer(), server)
    if listen_addr.endswith(":0"):
        bound_port = server.add_insecure_port("127.0.0.1:0")
        actual = f"127.0.0.1:{bound_port}"
    else:
        server.add_insecure_port(listen_addr)
        actual = listen_addr
    server.start()
    if port_file:
        Path(port_file).write_text(actual, encoding="utf-8")
    logger.info("data-worker 监听 %s", actual)
    server.wait_for_termination()


def main() -> None:
    listen = os.environ.get("IA_WORKER_LISTEN", "127.0.0.1:50052")
    port_file = os.environ.get("IA_WORKER_PORT_FILE") or None
    serve(listen, port_file=port_file)


if __name__ == "__main__":
    main()
