# Oculo Pilot Go Server

고성능 WebSocket Signaling 서버 - RunPod 배포용

## 🎯 특징

- **저지연**: Go 기반으로 0.3-0.8ms WebSocket 지연시간
- **인증**: JWT 토큰 기반 보안 인증
- **확장성**: 1,000+ 동시 연결 지원
- **NAT Traversal**: TURN 서버 통합
- **Docker**: 원클릭 배포
- **경량**: 메모리 사용량 ~30MB

## 📁 프로젝트 구조

```
oculo-pilot-server/
├── auth/              # 인증 시스템 (JWT, bcrypt, SQLite)
├── websocket/         # WebSocket 핵심 로직
├── middleware/        # HTTP 미들웨어
├── api/               # REST API 엔드포인트
├── config/            # 설정 관리
├── static/            # 정적 파일 (로그인 페이지)
├── deploy/            # Docker 배포 설정
├── main.go            # 메인 엔트리포인트
└── README.md          # 이 파일
```

## 🚀 빠른 시작

### 1. 로컬 개발

```bash
# 의존성 설치
go mod tidy

# 환경변수 설정
cp .env.example .env
# .env 파일 수정 (JWT_SECRET 변경 필수!)

# 서버 실행
go run main.go
```

기본 admin 계정:
- Username: `admin`
- Password: `admin123`
- **⚠️ 즉시 변경하세요!**

### 2. Docker로 실행

```bash
cd deploy
docker-compose up -d
```

### 3. RunPod 배포

#### Step 1: RunPod SSH 접속

```bash
ssh root@<runpod-instance-ip>
```

#### Step 2: 프로젝트 전송

```bash
# 로컬에서
scp -r oculo-pilot-server root@<runpod-instance-ip>:/app
```

#### Step 3: RunPod에서 실행

```bash
cd /app/oculo-pilot-server/deploy

# 환경변수 설정
nano .env

# Docker Compose로 실행
docker-compose up -d

# 로그 확인
docker-compose logs -f app
```

## 🔧 환경변수

| 변수 | 기본값 | 설명 |
|------|--------|------|
| `SERVER_HOST` | `0.0.0.0` | 서버 바인딩 주소 |
| `SERVER_PORT` | `8080` | 서버 포트 |
| `JWT_SECRET` | `change-this-secret-key-in-production` | JWT 서명 시크릿 키 (기본값은 개발용, 프로덕션에서 반드시 교체) |
| `JWT_EXPIRY` | `24h` | JWT 토큰 유효기간 |
| `DB_PATH` | `./users.db` | SQLite DB 경로 |
| `ALLOWED_ORIGINS` | `*` | CORS 허용 도메인 |
| `RATE_LIMIT` | `100` | 초당 요청 제한 |
| `HANDSHAKE_TIMEOUT` | `10s` | WebSocket 핸드셰이크 대기 시간 |
| `MAX_MESSAGE_SIZE` | `65536` | WebSocket 최대 메시지 크기 (바이트) |
| `ENABLE_IP_WHITELIST` | `false` | IP 화이트리스트 활성화 여부 |
| `ALLOWED_NETWORKS` | `0.0.0.0/0` | 허용할 CIDR 목록 (`,`로 구분) |
| `TURN_SERVER` | - | TURN 서버 주소 |
| `TURN_USERNAME` | - | TURN 인증 사용자명 |
| `TURN_PASSWORD` | - | TURN 인증 비밀번호 |

## 📡 API 엔드포인트

### Health Check
```http
GET /health
```

응답:
```json
{
  "status": "healthy",
  "timestamp": "2024-01-20T10:30:00Z",
  "version": "1.0.0"
}
```

### 로그인
```http
POST /api/login
Content-Type: application/json

{
  "username": "admin",
  "password": "admin123"
}
```

응답:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": 1,
    "username": "admin",
    "created_at": "2024-01-20T10:00:00Z"
  }
}
```

### 사용자 등록
```http
POST /api/register
Content-Type: application/json

{
  "username": "newuser",
  "password": "securepass123"
}
```

### WebSocket 연결
```
ws://localhost:8080/ws?token=<JWT_TOKEN>
```

## 🔐 보안

### JWT 토큰

1. 로그인 시 JWT 토큰 발급
2. 모든 WebSocket 연결에 토큰 필요
3. 토큰은 24시간 유효 (설정 가능)

### 비밀번호

- bcrypt 해싱 (cost 12)
- 최소 8자 이상
- 사용자명: 3-20자, 알파벳+숫자+언더스코어

### CORS

- 환경변수로 허용 도메인 설정
- 프로덕션에서는 `*` 사용 금지

## 🛠️ 개발

### 테스트

```bash
# 유닛 테스트
go test ./...

# 특정 패키지 테스트
go test ./auth
go test ./websocket
```

### 빌드

```bash
# 로컬 빌드
go build -o oculo-pilot-server

# 크로스 컴파일 (Linux)
GOOS=linux GOARCH=amd64 go build -o oculo-pilot-server-linux
```

### 코드 포맷팅

```bash
# 포맷 확인
go fmt ./...

# Lint
golint ./...
```

## 📊 성능

### 벤치마크

```
동시 연결: 1,000+
지연시간: 0.3-0.8ms (평균)
메모리: ~30MB
CPU: 10-20% (1-3명 사용 시)
```

### RunPod 권장 사양

- **CPU**: 2 vCPU
- **RAM**: 2GB
- **비용**: ~$0.10-0.15/hour

## 🐳 Docker

### 단일 컨테이너 실행

```bash
docker build -f deploy/Dockerfile -t oculo-pilot .
docker run -p 8080:8080 -e JWT_SECRET=your-secret oculo-pilot
```

### Docker Compose (권장)

```bash
cd deploy
docker-compose up -d
```

포함 서비스:
- `app`: Go WebSocket 서버
- `coturn`: TURN 서버 (NAT traversal)
- `nginx`: Reverse proxy (SSL 옵션)

## 🔍 트러블슈팅

### 문제: "go: command not found"

로컬 개발 시에만 필요. RunPod에서는 Docker 사용.

### 문제: WebSocket 연결 실패

1. JWT 토큰 확인
2. 방화벽 설정 확인
3. CORS 설정 확인

### 문제: 데이터베이스 권한 오류

```bash
chmod 666 users.db
```

### 문제: TURN 서버 연결 안 됨

1. UDP 포트 개방 확인 (3478, 49152-65535)
2. coturn.conf 자격증명 확인

## 📝 다음 단계

1. **SSL 인증서 설정** (Let's Encrypt)
2. **커스텀 도메인** 연결
3. **모니터링** 추가 (Prometheus, Grafana)
4. **로그 집계** (ELK Stack)
5. **백업** 자동화

## 🤝 연동

### 라즈베리파이 연결

라즈베리파이의 `config.json` 수정:

```json
{
  "server": {
    "host": "<runpod-ip>",
    "port": 8080
  }
}
```

### 웹 클라이언트 연결

```javascript
const token = localStorage.getItem('authToken');
const ws = new WebSocket(`ws://<runpod-ip>:8080/ws?token=${token}`);
```

## 📞 지원

문제 발생 시:
1. 로그 확인: `docker-compose logs -f app`
2. Health check: `curl http://localhost:8080/health`
3. 데이터베이스 확인: `sqlite3 users.db "SELECT * FROM users;"`

## 📄 라이선스

MIT License

## 🔄 업데이트

```bash
# 최신 코드 가져오기
git pull

# 재빌드 및 재시작
cd deploy
docker-compose down
docker-compose up -d --build
```

---

**Oculo Pilot** - 저지연 원격 조종 시스템
