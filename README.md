## Пример использования docker-compose и godog для интеграционного тестирования микросервисов

### Дисклеймер
1. Это пример, а не single source of truth: меняйте, экспериментируйте, не копируйте бездумно.
2. Тестирование с использованием RabbitMQ не самый удобный вариант.
Гораздо проще добавить необходимых ручек в сервис или делать прямой запрос в базу.
Все зависит от того, какие части системы вы хотите покрыть тестами.
3. В данном примере в тестах нет работы с БД (например, очистка), но, я думаю,
вам не составит труда сделать её при необходимости.
4. В сервисах специально мало информации об ошибках, чтобы в случае таковых,
вы сами копались в коде и меняли его.
5. Здесь godog запускается из кода, но его можно вызывать как и самостоятельную тулзу
(будьте готовы ответить, почему здесь именно так :).
6. В примере инициализация начальной структуры базы и кролика выполняется через Docker.
В принципе вы можете делать это через код тестов. Просто по моему мнению это все-таки больше инфраструктурная задача
и решил показать такой вариант.

Все команды выполняются из корня проекта.

### Запуск микросервисов
Поднимаем docker-compose
```shell script
$ make up
docker-compose up -d --build
Creating network "godog_example_db" with driver "bridge"
Creating network "godog_example_rabbit" with driver "bridge"
Building notification_service
...
Successfully built 8fc475a1a227
Successfully tagged godog_example_notification_service:latest
Building registration_service
...
Successfully built 4d33fe59d777
Successfully tagged godog_example_registration_service:latest
Creating godog_example_rabbit_1   ... done
Creating godog_example_postgres_1 ... done
Creating godog_example_notification_service_1 ... done
Creating godog_example_registration_service_1 ... done
```
```shell script
$ docker-compose ps
                Name                              Command               State                                             Ports                                           
--------------------------------------------------------------------------------------------------------------------------------------------------------------------------
godog_example_notification_service_1   /bin/notify_service              Up                                                                                                
godog_example_postgres_1               docker-entrypoint.sh postgres    Up      0.0.0.0:5432->5432/tcp                                                                    
godog_example_rabbit_1                 docker-entrypoint.sh rabbi ...   Up      15671/tcp, 0.0.0.0:15672->15672/tcp, 25672/tcp, 4369/tcp, 5671/tcp, 0.0.0.0:5672->5672/tcp
godog_example_registration_service_1   /bin/reg_service                 Up      0.0.0.0:8088->8088/tcp 
```
Проверяем доступность сервиса регистрации
```shell script
$ curl http://localhost:8088/
OK
```
Регистрируем пользователя
```shell script
$ curl -d '{"first_name":"otus", "email":"otus@otus.ru", "age": 27}' -H "Content-Type: application/json" -X POST http://localhost:8088/api/v1/registration
```
Проверяем, что в базе появился пользователь
```shell script
$ docker exec postgres psql -U test -d exampledb -c "select * from users;"
 first_name |    email     | age 
------------+--------------+-----
 otus       | otus@otus.ru |  27
(1 row)
```
Проверяем, что было опубликовано событие о новой регистрации
![User registration event](images/user_reg_event.png)

**Теперь у нас есть возможность писать тесты и дебажить их локально,
так как вся инфраструктура поднята в Docker, а необходимые порты пробросаны на host.**
Необходимо только помнить, куда тесты ходят - на localhost или во внутреннюю сеть докера.

Останавливаем docker-compose
```shell script
$ make stop
docker-compose down
Stopping godog_example_registration_service_1 ... done
Stopping godog_example_notification_service_1 ... done
Stopping godog_example_rabbit_1               ... done
Stopping godog_example_postgres_1             ... done
Removing godog_example_registration_service_1 ... done
Removing godog_example_notification_service_1 ... done
Removing godog_example_rabbit_1               ... done
Removing godog_example_postgres_1             ... done
Removing network godog_example_db
Removing network godog_example_rabbit
```

### Интеграционное тестирование
После разработки и отладки тестов проверяем их работу из контейнера.
Запускаем тесты
```shell script
$ make test
...
Creating network "godog_example_db" with driver "bridge"
Creating network "godog_example_rabbit" with driver "bridge"
Building notification_service
...
Successfully built 8fc475a1a227
Successfully tagged godog_example_notification_service:latest
Building registration_service
...
Successfully built 4d33fe59d777
Successfully tagged godog_example_registration_service:latest
Building integration_tests
...
Successfully built f674e81ce394
Successfully tagged godog_example_integration_tests:latest

Creating godog_example_rabbit_1   ... done
Creating godog_example_postgres_1 ... done
Creating godog_example_notification_service_1 ... done
Creating godog_example_registration_service_1 ... done
Creating godog_example_integration_tests_1    ... done

Starting godog_example_postgres_1 ... done
Starting godog_example_rabbit_1   ... done
Starting godog_example_registration_service_1 ... done
Starting godog_example_notification_service_1 ... done

Wait 5s for service availability...
...... 6
2 scenarios (2 passed)
6 steps (6 passed)
3.0577195s
testing: warning: no tests to run
PASS
ok      godog_example/integration_tests 8.101s

Stopping godog_example_registration_service_1 ... done
Stopping godog_example_notification_service_1 ... done
Stopping godog_example_postgres_1             ... done
Stopping godog_example_rabbit_1               ... done

Removing godog_example_integration_tests_run_45230d1f037c ... done
Removing godog_example_integration_tests_1                ... done
Removing godog_example_registration_service_1             ... done
Removing godog_example_notification_service_1             ... done
Removing godog_example_postgres_1                         ... done
Removing godog_example_rabbit_1                           ... done
Removing network godog_example_db
Removing network godog_example_rabbit

$ echo $?
0
```
В логе мы видим:
- поднимаются микросервисы;
- успешно выполняются 2 тестовых сценария из 6 шагов;
- контейнеры останавливаются и удаляются;
- **сам скрипт возвращает 0 и 1 в зависимости от статуса прохождения тестов**
(это важно, так пригодится нам в Continuous Integration).
