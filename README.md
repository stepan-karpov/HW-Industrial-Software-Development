### **Домашнее задание: Развёртывание распределённой системы логирования и хранения с резервным копированием**

### Сборка и запуск приложения

#### Локальная сборка (Go)

Приложение читает конфиг только из `/app/config/config.json` (как в `k8s/configmap.yaml`). Удобнее проверять через Docker или Kubernetes ниже.

```bash
cd app
go build -o app .
```

Проверка ручек:

```bash
curl http://127.0.0.1:8280/
curl http://127.0.0.1:8280/status
curl -X POST http://127.0.0.1:8280/log -H 'Content-Type: application/json' -d '{"message":"test"}'
curl http://127.0.0.1:8280/logs
```

#### Сборка Docker-образа

```bash
docker build -t app:latest ./app
```

#### Локальный Kubernetes (kind)

Если кластера ещё нет:

```bash
kind create cluster --name hw-app
kubectl config use-context kind-hw-app
```

Собрать образ, загрузить в kind, применить манифесты:

```bash
docker build -t app:latest ./app
kind load docker-image app:latest --name hw-app
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/pod.yaml
```

Логи (`/app/logs/app.log`) монтируются через **`hostPath`** на узле (`/mnt/k8s-app-logs` на ноде → `/app/logs` в контейнере). Пока все реплики живут **на одном узле** (типичный **kind** с одной нодой), это **один и тот же файл** для всех подов. В **мульти-нодовом** кластере у каждой ноды свой диск — нужен общий том (**NFS**, **ReadWriteMany** PVC и т.п.), иначе логи снова разъедутся по узлам.

Приложение в контейнере слушает **8280** (порт **8080** на хосте остаётся свободным для других процессов). У `kind` проброс `hostPort` на машину-хост часто недоступен, поэтому надёжнее:

```bash
kubectl port-forward pod/app-pod 8280:8280
```

Проверка:

```bash
curl http://127.0.0.1:8280/status
```

1. **Создать пользовательское веб-приложение (API)**  
   Приложение должно реализовать следующие REST-эндпоинты:
   - `GET /` — возвращает строку `"Welcome to the custom app"`
   - `GET /status` — возвращает JSON `{"status": "ok"}`
   - `POST /log` — принимает JSON `{"message": "some log"}` и записывает его в файл `/app/logs/app.log`
   - `GET /logs` — возвращает содержимое файла `/app/logs/app.log`

   Приложение должно:
   - Писать логи в `/app/logs/app.log`
   - Использовать конфигурационные параметры (например, уровень логирования, порт, заголовок приветствия) из ConfigMap

2. **Развернуть приложение как Pod для начального теста**
   - Написать Dockerfile для приложения
   - Создать Pod, монтирующий:
      - `emptyDir` volume в `/app/logs`
      - ConfigMap с настройками в `/app/config` (или через переменные окружения)

3. **Развернуть приложение как Deployment**
   - Создать Deployment с 3 репликами
   - Настроить монтирование `emptyDir` для логов
   - Обновить Deployment, чтобы изменения в ConfigMap автоматически применялись
   - Проверить через Service и `kubectl port-forward`, что API работает

4. **Создать Service для балансировки нагрузки**
   - ClusterIP-сервис, направляющий трафик на поды приложения
   - Проверить: `curl http://<service-name>/logs` и `curl -X POST http://<service-name>/log -d '{"message": "test"}'`
   - Убедиться, что запросы распределяются между подами

5. **Развернуть DaemonSet с log-agent**
   - DaemonSet должен:
      - Быть запущен на каждом узле
      - Собирать логи приложения из подов (через `hostPath` или `emptyDir`, при наличии доступа)
      - Перенаправлять логи во stdout или сохранять локально на узле
   - Проверить, что `kubectl logs <log-agent-pod>` содержит записи из `app.log`

6. **Развернуть CronJob для архивирования логов**
   - CronJob должен запускаться раз в 10 минут
   - Команда: `tar -czf /tmp/app-logs-<timestamp>.tar.gz /app/logs/`
   - Логи берутся с сервисов приложения через HTTP API `/logs` (например, `curl`) или из общей директории, если доступна
   - Результат сохраняется в контейнере в `/tmp` (внутри пода CronJob)

7. **Создать единый bash-скрипт `deploy.sh` для автоматического развёртывания всей системы**
   - Скрипт должен:
      - Создавать все необходимые **ConfigMap**, **Pod**, **Deployment**, **Service**, **DaemonSet**, **StatefulSet**, **CronJob** и другие объекты
      - Использовать команды `kubectl apply -f` с заранее подготовленными YAML-файлами
      - Ожидать готовности ключевых компонентов
   - В **README.md** проекта добавьте команду для запуска скрипта из терминала