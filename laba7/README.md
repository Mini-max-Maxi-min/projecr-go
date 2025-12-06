

# **ЛАБОРАТОРНА РОБОТА №7**

## Тема: Оркестрація мікросервісів

## Мета: Навчитися керувати кількома контейнерами за допомогою Docker Compose та продемонструвати взаємодію мікросервісів у розподіленій системі.



# **1. Короткі теоретичні відомості**

Оркестрація мікросервісів — це процес автоматизованого управління контейнерами, їхнім життєвим циклом, масштабуванням, оновленнями та відновленням після збоїв. Вона дозволяє координувати роботу великої кількості сервісів у розподіленій програмній системі.

Основні інструменти:

### **Docker Compose**

Використовується для локальної оркестрації. Дозволяє:

* запускати багато сервісів однією командою;
* створювати мережі між контейнерами;
* визначати залежності, томи, змінні середовища.

### **Kubernetes**

Більш масштабне рішення, яке підтримує:

* автоматичне масштабування (Horizontal Pod Autoscaler),
* відмовостійкість,
* оновлення без простоїв (rolling updates),
* балансування навантаження.

В навчальних проєктах зазвичай використовується Docker Compose, а у продакшені — Kubernetes.



# **2. Завдання**

1. Розробити `docker-compose.yml` для запуску кількох сервісів.
2. Налаштувати мережеву взаємодію між контейнерами.
3. Перевірити роботу системи через REST або інші API.

У нашому варіанті — створити Telegram-бота, який працює з:

* PostgreSQL,
* Kafka producer/consumer,
* власним мікросервісом (ботом).

Усі ці сервіси повинні бути запущені через Docker Compose.


# **3. Протоколи виконання**

## **3.1. Створення мікросервісу Telegram-бота**

Нижче представлено повний вихідний код мікросервісу, який:

* працює з PostgreSQL;
* використовує Kafka як брокер подій;
* обробляє Telegram-команди;
* зберігає, видаляє та показує вправи для Workout Tracker.

---



# **3.2. Docker Compose (сервіси: bot + PostgreSQL + Kafka + Zookeeper)**

```yaml
version: "3.9"

services:

  bot:
    build: .
    container_name: workout_bot
    depends_on:
      - db
      - kafka
    environment:
      TELEGRAM_TOKEN: "${TELEGRAM_TOKEN}"
      DATABASE_URL: "postgres://postgres:postgres@db:5432/workouts?sslmode=disable"
      KAFKA_BROKER: "kafka:29092"
      KAFKA_TOPIC: "workout-events"
    restart: always

  db:
    image: postgres:15
    container_name: workouts_db
    environment:
      POSTGRES_PASSWORD: postgres
      POSTGRES_USER: postgres
      POSTGRES_DB: workouts
    ports:
      - "5432:5432"
    restart: always

  zookeeper:
    image: confluentinc/cp-zookeeper:latest
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181
    restart: always

  kafka:
    image: confluentinc/cp-kafka:latest
    depends_on:
      - zookeeper
    ports:
      - "9092:9092"
    environment:
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://kafka:29092,PLAINTEXT_HOST://localhost:9092
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: PLAINTEXT:PLAINTEXT,PLAINTEXT_HOST:PLAINTEXT
      KAFKA_INTER_BROKER_LISTENER_NAME: PLAINTEXT
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
    restart: always
```

Запуск:


docker compose up -d --build






Результат — повноцінна розподілена система, яка використовує декілька сервісів та працює як цілісний застосунок.

---

# . Контрольні запитання

1. Що таке оркестрація контейнерів?
   Автоматизоване управління запуском, масштабуванням та життєвим циклом контейнерів.

2. Основні елементи Docker Compose?
   Services, networks, volumes, environment, depends_on.

3. Основні компоненти Kubernetes?
   Pod, Deployment, Service, Ingress, ConfigMap, Secret.

4. Як забезпечується масштабування у Kubernetes?
   Через Horizontal Pod Autoscaler.

5. Що таке Helm і для чого його використовують?
   Пакетний менеджер для Kubernetes; спрощує деплой складних систем.

6. Як реалізується стійкість до відмов у Kubernetes?
   Автоматичним перезапуском Pod-ів та перенесенням їх на інші вузли.

7. Як організувати спільну мережу для кількох контейнерів?
   Через Docker Compose networks або Kubernetes Service.
