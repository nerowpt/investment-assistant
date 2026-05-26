"""启动 data-worker gRPC 服务。"""

import sys

print("[data-worker] boot", flush=True)

from data_worker.server import main

if __name__ == "__main__":
    main()
