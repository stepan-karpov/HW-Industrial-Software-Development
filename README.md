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

#### Пункт 3: Deployment (3 реплики) и проверка через Service

Перед применением Deployment лучше убрать одиночный Pod из пункта 2, чтобы не путаться:

```bash
kubectl delete pod app-pod --ignore-not-found
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
kubectl rollout status deployment/app --timeout=120s
```

В Deployment для логов используется **`hostPath`** [`/mnt/k8s-app-logs`](k8s/deployment.yaml) — общий на узле каталог (как в [`k8s/pod.yaml`](k8s/pod.yaml)), чтобы **DaemonSet log-agent** мог читать тот же `app.log` (п. 5).

Чтобы при изменении `ConfigMap` поды **пересоздались** и подхватили конфиг, в `k8s/deployment.yaml` задана аннотация **`checksum/config`**. После правки `k8s/configmap.yaml` обновите значение checksum, например так (нужен работающий кластер и применённый ConfigMap):

```bash
kubectl get configmap app-config -o jsonpath='{.data.config\.json}' | sha256sum | awk '{print $1}'
```

Подставьте полученную строку в `checksum/config` в `k8s/deployment.yaml` и снова выполните `kubectl apply -f k8s/deployment.yaml`. Альтернатива без смены checksum: `kubectl rollout restart deployment/app` (принудительный перезапуск).

Проверка API через **Service** и `port-forward`:

```bash
kubectl port-forward svc/app 8280:8280
curl http://127.0.0.1:8280/status
```

#### Пункт 4: ClusterIP Service и балансировка

После изменений в коде приложения пересоберите образ и обновите поды, например:

```bash
docker build -t app:latest ./app
kind load docker-image app:latest --name hw-app
kubectl rollout restart deployment/app
```

Сервис описан в [`k8s/service.yaml`](k8s/service.yaml): имя **`app`**, тип **`ClusterIP`**, порт **8280** → поды с лейблом `app: app`. Внутри кластера DNS: **`http://app.default.svc.cluster.local:8280`** (в том же namespace достаточно **`http://app:8280`**).

Проверка ручек через Service с хоста (после `kubectl port-forward svc/app 8280:8280`):

```bash
curl -s http://127.0.0.1:8280/logs
curl -s -X POST http://127.0.0.1:8280/log -H 'Content-Type: application/json' -d '{"message":"test"}'
```

Ответ **`GET /status`** содержит поле **`pod`** (имя пода в Kubernetes), чтобы увидеть балансировку.

**Почему через `kubectl port-forward svc/app` часто виден только один `pod`:** прокси `kubectl` к **Service** не обязан вести себя как полноценный kube-proxy: трафик может уходить **на один выбранный под** на всё время сессии `port-forward`, даже если с хоста открываются новые TCP‑соединения и даже с `Connection: close`. Это ожидаемое ограничение, а не ошибка Service.

**Надёжная проверка балансировки** — запросы к **`ClusterIP` Service изнутри кластера** (там kube-proxy реально распределяет соединения между подами):

```bash
kubectl run curl-lb --rm --restart=Never -i --image=curlimages/curl -- \
  sh -c 'for i in $(seq 1 30); do curl -s -H "Connection: close" http://app.default.svc.cluster.local:8280/status; echo; done' \
  | grep -oE '"pod":"[^"]+"' | sort | uniq -c
```

(Под и namespace по умолчанию — `default`; короткое имя **`http://app:8280`** тоже подойдёт.)

С хоста можно оставить цикл с **`Connection: close`** — иногда появятся разные `pod`, но для отчёта лучше опираться на команду выше.

#### Пункт 5: DaemonSet `log-agent`

Манифест: [`k8s/daemonset-log-agent.yaml`](k8s/daemonset-log-agent.yaml). На каждом узле запускается контейнер **`busybox`**, который монтирует тот же **`hostPath`** `/mnt/k8s-app-logs`, что и приложение, и пишет содержимое **`app.log`** в **stdout** (`tail -F`).

```bash
kubectl apply -f k8s/daemonset-log-agent.yaml
kubectl rollout status daemonset/log-agent --timeout=120s
```

Сгенерируйте строки в логе приложения (например, `POST /log`), затем:

```bash
kubectl logs -l app=log-agent --tail=50
```

В выводе должны быть строки из **`/app/logs/app.log`** (время и сообщения). Если подов несколько, смотрите лог конкретного: `kubectl get pods -l app=log-agent` и `kubectl logs <имя-пода>`.

#### Пункт 6: CronJob архивирования логов

Манифест: [`k8s/cronjob-archive-logs.yaml`](k8s/cronjob-archive-logs.yaml). Расписание **каждые 10 минут** (`*/10 * * * *`). Контейнер **`alpine`** монтирует тот же **`hostPath`** `/mnt/k8s-app-logs` в **`/app/logs`** (как у приложения) и выполняет:

`tar -czf /tmp/app-logs-<timestamp>.tar.gz /app/logs`

Архивы лежат в **`/tmp`** внутри пода Job (по заданию). Вместо `hostPath` можно было бы перед архивацией скачать логи с **`curl http://app:8280/logs`** — здесь используется общая директория на узле, как в задании. На **мульти-нодовом** кластере убедитесь, что Job попадает на узел с данными (или используйте общий том / HTTP).

```bash
kubectl apply -f k8s/cronjob-archive-logs.yaml
```

Не дожидаясь 10 минут, запустите Job вручную:

```bash
kubectl create job --from=cronjob/app-logs-archive app-logs-archive-manual
kubectl wait --for=condition=complete job/app-logs-archive-manual --timeout=120s
kubectl logs job/app-logs-archive-manual
```

Проверка архива в поде Job:

```bash
POD=$(kubectl get pods -l job-name=app-logs-archive-manual -o jsonpath='{.items[0].metadata.name}')
kubectl exec "$POD" -- ls -la /tmp
kubectl exec "$POD" -- sh -c 'tar -tzf /tmp/app-logs-*.tar.gz'
```

1. **Создать пользовательское веб-приложение (API)**  
   Приложение должно реализовать следующие REST-эндпоинты:
   - `GET /` — возвращает строку `"Welcome to the custom app"`
   - `GET /status` — возвращает JSON `{"status": "ok", "pod": "<имя пода>"}` (поле `pod` нужно для проверки балансировки в п. 4)
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

### Запуск развёртывания одной командой

```bash
bash deploy.sh
```