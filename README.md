# ntun
Туннель с разными протоколами транспорта и входящими/исходящими подключениями

### Установка
После клонирования репозитория в папке
```
yarn
```
Создать конфинг, например в `./configs/config.yaml`

Запуск
```
node ./src/ntun.cli.js ./configs/config.yaml
```

### Пример конфига клиента

Запускается socks5 сервер на порту 8080, данные передаются по протоколу WebSockets на сервер `ws://MY_HOST:port`, с ограничением скорости 0.5 mbps

```yaml
input: 
  type: socks5
  port: 8080
transport: 
  type: ws
  host: MY_HOST
  port: 8081
  rateLimit: 0.5mbps

```

### Пример конфига сервера

Запускается ws сервер на 0.0.0.0:8081 для транспорта данных, далее все подключения уходят свободно в интернет

```yaml
output: 
  type: direct
transport: 
  type: ws
  host: 0.0.0.0
  port: 8081
  rateLimit: 0.5mbps
```
