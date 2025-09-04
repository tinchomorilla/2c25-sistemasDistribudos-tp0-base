# TP0: Docker + Comunicaciones + Concurrencia

En el presente repositorio se provee un esqueleto básico de cliente/servidor, en donde todas las dependencias del mismo se encuentran encapsuladas en containers. Los alumnos deberán resolver una guía de ejercicios incrementales, teniendo en cuenta las condiciones de entrega descritas al final de este enunciado.

El cliente (Golang) y el servidor (Python) fueron desarrollados en diferentes lenguajes simplemente para mostrar cómo dos lenguajes de programación pueden convivir en el mismo proyecto con la ayuda de containers, en este caso utilizando [Docker Compose](https://docs.docker.com/compose/).

## Instrucciones de uso

El repositorio cuenta con un **Makefile** que incluye distintos comandos en forma de targets. Los targets se ejecutan mediante la invocación de: **make \<target\>**. Los target imprescindibles para iniciar y detener el sistema son **docker-compose-up** y **docker-compose-down**, siendo los restantes targets de utilidad para el proceso de depuración.

Los targets disponibles son:

| target                | accion                                                                                                                                                                                                                                                                                                                                                                |
| --------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `docker-compose-up`   | Inicializa el ambiente de desarrollo. Construye las imágenes del cliente y el servidor, inicializa los recursos a utilizar (volúmenes, redes, etc) e inicia los propios containers.                                                                                                                                                                                   |
| `docker-compose-down` | Ejecuta `docker-compose stop` para detener los containers asociados al compose y luego `docker-compose down` para destruir todos los recursos asociados al proyecto que fueron inicializados. Se recomienda ejecutar este comando al finalizar cada ejecución para evitar que el disco de la máquina host se llene de versiones de desarrollo y recursos sin liberar. |
| `docker-compose-logs` | Permite ver los logs actuales del proyecto. Acompañar con `grep` para lograr ver mensajes de una aplicación específica dentro del compose.                                                                                                                                                                                                                            |
| `docker-image`        | Construye las imágenes a ser utilizadas tanto en el servidor como en el cliente. Este target es utilizado por **docker-compose-up**, por lo cual se lo puede utilizar para probar nuevos cambios en las imágenes antes de arrancar el proyecto.                                                                                                                       |
| `build`               | Compila la aplicación cliente para ejecución en el _host_ en lugar de en Docker. De este modo la compilación es mucho más veloz, pero requiere contar con todo el entorno de Golang y Python instalados en la máquina _host_.                                                                                                                                         |

### Servidor

Se trata de un "echo server", en donde los mensajes recibidos por el cliente se responden inmediatamente y sin alterar.

Se ejecutan en bucle las siguientes etapas:

1. Servidor acepta una nueva conexión.
2. Servidor recibe mensaje del cliente y procede a responder el mismo.
3. Servidor desconecta al cliente.
4. Servidor retorna al paso 1.

### Cliente

se conecta reiteradas veces al servidor y envía mensajes de la siguiente forma:

1. Cliente se conecta al servidor.
2. Cliente genera mensaje incremental.
3. Cliente envía mensaje al servidor y espera mensaje de respuesta.
4. Servidor responde al mensaje.
5. Servidor desconecta al cliente.
6. Cliente verifica si aún debe enviar un mensaje y si es así, vuelve al paso 2.

### Ejemplo

Al ejecutar el comando `make docker-compose-up` y luego `make docker-compose-logs`, se observan los siguientes logs:

```
client1  | 2024-08-21 22:11:15 INFO     action: config | result: success | client_id: 1 | server_address: server:12345 | loop_amount: 5 | loop_period: 5s | log_level: DEBUG
client1  | 2024-08-21 22:11:15 INFO     action: receive_message | result: success | client_id: 1 | msg: [CLIENT 1] Message N°1
server   | 2024-08-21 22:11:14 DEBUG    action: config | result: success | port: 12345 | listen_backlog: 5 | logging_level: DEBUG
server   | 2024-08-21 22:11:14 INFO     action: accept_connections | result: in_progress
server   | 2024-08-21 22:11:15 INFO     action: accept_connections | result: success | ip: 172.25.125.3
server   | 2024-08-21 22:11:15 INFO     action: receive_message | result: success | ip: 172.25.125.3 | msg: [CLIENT 1] Message N°1
server   | 2024-08-21 22:11:15 INFO     action: accept_connections | result: in_progress
server   | 2024-08-21 22:11:20 INFO     action: accept_connections | result: success | ip: 172.25.125.3
server   | 2024-08-21 22:11:20 INFO     action: receive_message | result: success | ip: 172.25.125.3 | msg: [CLIENT 1] Message N°2
server   | 2024-08-21 22:11:20 INFO     action: accept_connections | result: in_progress
client1  | 2024-08-21 22:11:20 INFO     action: receive_message | result: success | client_id: 1 | msg: [CLIENT 1] Message N°2
server   | 2024-08-21 22:11:25 INFO     action: accept_connections | result: success | ip: 172.25.125.3
server   | 2024-08-21 22:11:25 INFO     action: receive_message | result: success | ip: 172.25.125.3 | msg: [CLIENT 1] Message N°3
client1  | 2024-08-21 22:11:25 INFO     action: receive_message | result: success | client_id: 1 | msg: [CLIENT 1] Message N°3
server   | 2024-08-21 22:11:25 INFO     action: accept_connections | result: in_progress
server   | 2024-08-21 22:11:30 INFO     action: accept_connections | result: success | ip: 172.25.125.3
server   | 2024-08-21 22:11:30 INFO     action: receive_message | result: success | ip: 172.25.125.3 | msg: [CLIENT 1] Message N°4
server   | 2024-08-21 22:11:30 INFO     action: accept_connections | result: in_progress
client1  | 2024-08-21 22:11:30 INFO     action: receive_message | result: success | client_id: 1 | msg: [CLIENT 1] Message N°4
server   | 2024-08-21 22:11:35 INFO     action: accept_connections | result: success | ip: 172.25.125.3
server   | 2024-08-21 22:11:35 INFO     action: receive_message | result: success | ip: 172.25.125.3 | msg: [CLIENT 1] Message N°5
client1  | 2024-08-21 22:11:35 INFO     action: receive_message | result: success | client_id: 1 | msg: [CLIENT 1] Message N°5
server   | 2024-08-21 22:11:35 INFO     action: accept_connections | result: in_progress
client1  | 2024-08-21 22:11:40 INFO     action: loop_finished | result: success | client_id: 1
client1 exited with code 0
```

## Parte 1: Introducción a Docker

En esta primera parte del trabajo práctico se plantean una serie de ejercicios que sirven para introducir las herramientas básicas de Docker que se utilizarán a lo largo de la materia. El entendimiento de las mismas será crucial para el desarrollo de los próximos TPs.

### Ejercicio N°1:

Definir un script de bash `generar-compose.sh` que permita crear una definición de Docker Compose con una cantidad configurable de clientes. El nombre de los containers deberá seguir el formato propuesto: client1, client2, client3, etc.

El script deberá ubicarse en la raíz del proyecto y recibirá por parámetro el nombre del archivo de salida y la cantidad de clientes esperados:

`./generar-compose.sh docker-compose-dev.yaml 5`

Considerar que en el contenido del script pueden invocar un subscript de Go o Python:

```
#!/bin/bash
echo "Nombre del archivo de salida: $1"
echo "Cantidad de clientes: $2"
python3 mi-generador.py $1 $2
```

En el archivo de Docker Compose de salida se pueden definir volúmenes, variables de entorno y redes con libertad, pero recordar actualizar este script cuando se modifiquen tales definiciones en los sucesivos ejercicios.

### Ejercicio N°2:

Modificar el cliente y el servidor para lograr que realizar cambios en el archivo de configuración no requiera reconstruír las imágenes de Docker para que los mismos sean efectivos. La configuración a través del archivo correspondiente (`config.ini` y `config.yaml`, dependiendo de la aplicación) debe ser inyectada en el container y persistida por fuera de la imagen (hint: `docker volumes`).

### Ejercicio N°3:

Crear un script de bash `validar-echo-server.sh` que permita verificar el correcto funcionamiento del servidor utilizando el comando `netcat` para interactuar con el mismo. Dado que el servidor es un echo server, se debe enviar un mensaje al servidor y esperar recibir el mismo mensaje enviado.

En caso de que la validación sea exitosa imprimir: `action: test_echo_server | result: success`, de lo contrario imprimir:`action: test_echo_server | result: fail`.

El script deberá ubicarse en la raíz del proyecto. Netcat no debe ser instalado en la máquina _host_ y no se pueden exponer puertos del servidor para realizar la comunicación (hint: `docker network`). `

### Ejercicio N°4:

Modificar servidor y cliente para que ambos sistemas terminen de forma _graceful_ al recibir la signal SIGTERM. Terminar la aplicación de forma _graceful_ implica que todos los _file descriptors_ (entre los que se encuentran archivos, sockets, threads y procesos) deben cerrarse correctamente antes que el thread de la aplicación principal muera. Loguear mensajes en el cierre de cada recurso (hint: Verificar que hace el flag `-t` utilizado en el comando `docker compose down`).

## Parte 2: Repaso de Comunicaciones

Las secciones de repaso del trabajo práctico plantean un caso de uso denominado **Lotería Nacional**. Para la resolución de las mismas deberá utilizarse como base el código fuente provisto en la primera parte, con las modificaciones agregadas en el ejercicio 4.

### Ejercicio N°5:

Modificar la lógica de negocio tanto de los clientes como del servidor para nuestro nuevo caso de uso.

#### Cliente

Emulará a una _agencia de quiniela_ que participa del proyecto. Existen 5 agencias. Deberán recibir como variables de entorno los campos que representan la apuesta de una persona: nombre, apellido, DNI, nacimiento, numero apostado (en adelante 'número'). Ej.: `NOMBRE=Santiago Lionel`, `APELLIDO=Lorca`, `DOCUMENTO=30904465`, `NACIMIENTO=1999-03-17` y `NUMERO=7574` respectivamente.

Los campos deben enviarse al servidor para dejar registro de la apuesta. Al recibir la confirmación del servidor se debe imprimir por log: `action: apuesta_enviada | result: success | dni: ${DNI} | numero: ${NUMERO}`.

#### Servidor

Emulará a la _central de Lotería Nacional_. Deberá recibir los campos de la cada apuesta desde los clientes y almacenar la información mediante la función `store_bet(...)` para control futuro de ganadores. La función `store_bet(...)` es provista por la cátedra y no podrá ser modificada por el alumno.
Al persistir se debe imprimir por log: `action: apuesta_almacenada | result: success | dni: ${DNI} | numero: ${NUMERO}`.

#### Comunicación:

Se deberá implementar un módulo de comunicación entre el cliente y el servidor donde se maneje el envío y la recepción de los paquetes, el cual se espera que contemple:

- Definición de un protocolo para el envío de los mensajes.
- Serialización de los datos.
- Correcta separación de responsabilidades entre modelo de dominio y capa de comunicación.
- Correcto empleo de sockets, incluyendo manejo de errores y evitando los fenómenos conocidos como [_short read y short write_](https://cs61.seas.harvard.edu/site/2018/FileDescriptors/).

### Ejercicio N°6:

Modificar los clientes para que envíen varias apuestas a la vez (modalidad conocida como procesamiento por _chunks_ o _batchs_).
Los _batchs_ permiten que el cliente registre varias apuestas en una misma consulta, acortando tiempos de transmisión y procesamiento.

La información de cada agencia será simulada por la ingesta de su archivo numerado correspondiente, provisto por la cátedra dentro de `.data/datasets.zip`.
Los archivos deberán ser inyectados en los containers correspondientes y persistido por fuera de la imagen (hint: `docker volumes`), manteniendo la convencion de que el cliente N utilizara el archivo de apuestas `.data/agency-{N}.csv` .

En el servidor, si todas las apuestas del _batch_ fueron procesadas correctamente, imprimir por log: `action: apuesta_recibida | result: success | cantidad: ${CANTIDAD_DE_APUESTAS}`. En caso de detectar un error con alguna de las apuestas, debe responder con un código de error a elección e imprimir: `action: apuesta_recibida | result: fail | cantidad: ${CANTIDAD_DE_APUESTAS}`.

La cantidad máxima de apuestas dentro de cada _batch_ debe ser configurable desde config.yaml. Respetar la clave `batch: maxAmount`, pero modificar el valor por defecto de modo tal que los paquetes no excedan los 8kB.

Por su parte, el servidor deberá responder con éxito solamente si todas las apuestas del _batch_ fueron procesadas correctamente.

### Ejercicio N°7:

Modificar los clientes para que notifiquen al servidor al finalizar con el envío de todas las apuestas y así proceder con el sorteo.
Inmediatamente después de la notificacion, los clientes consultarán la lista de ganadores del sorteo correspondientes a su agencia.
Una vez el cliente obtenga los resultados, deberá imprimir por log: `action: consulta_ganadores | result: success | cant_ganadores: ${CANT}`.

El servidor deberá esperar la notificación de las 5 agencias para considerar que se realizó el sorteo e imprimir por log: `action: sorteo | result: success`.
Luego de este evento, podrá verificar cada apuesta con las funciones `load_bets(...)` y `has_won(...)` y retornar los DNI de los ganadores de la agencia en cuestión. Antes del sorteo no se podrán responder consultas por la lista de ganadores con información parcial.

Las funciones `load_bets(...)` y `has_won(...)` son provistas por la cátedra y no podrán ser modificadas por el alumno.

No es correcto realizar un broadcast de todos los ganadores hacia todas las agencias, se espera que se informen los DNIs ganadores que correspondan a cada una de ellas.

## Parte 3: Repaso de Concurrencia

En este ejercicio es importante considerar los mecanismos de sincronización a utilizar para el correcto funcionamiento de la persistencia.

### Ejercicio N°8:

Modificar el servidor para que permita aceptar conexiones y procesar mensajes en paralelo. En caso de que el alumno implemente el servidor en Python utilizando _multithreading_, deberán tenerse en cuenta las [limitaciones propias del lenguaje](https://wiki.python.org/moin/GlobalInterpreterLock).

## Condiciones de Entrega

Se espera que los alumnos realicen un _fork_ del presente repositorio para el desarrollo de los ejercicios y que aprovechen el esqueleto provisto tanto (o tan poco) como consideren necesario.

Cada ejercicio deberá resolverse en una rama independiente con nombres siguiendo el formato `ej${Nro de ejercicio}`. Se permite agregar commits en cualquier órden, así como crear una rama a partir de otra, pero al momento de la entrega deberán existir 8 ramas llamadas: ej1, ej2, ..., ej7, ej8.
(hint: verificar listado de ramas y últimos commits con `git ls-remote`)

Se espera que se redacte una sección del README en donde se indique cómo ejecutar cada ejercicio y se detallen los aspectos más importantes de la solución provista, como ser el protocolo de comunicación implementado (Parte 2) y los mecanismos de sincronización utilizados (Parte 3).

Se proveen [pruebas automáticas](https://github.com/7574-sistemas-distribuidos/tp0-tests) de caja negra. Se exige que la resolución de los ejercicios pase tales pruebas, o en su defecto que las discrepancias sean justificadas y discutidas con los docentes antes del día de la entrega. El incumplimiento de las pruebas es condición de desaprobación, pero su cumplimiento no es suficiente para la aprobación. Respetar las entradas de log planteadas en los ejercicios, pues son las que se chequean en cada uno de los tests.

La corrección personal tendrá en cuenta la calidad del código entregado y casos de error posibles, se manifiesten o no durante la ejecución del trabajo práctico. Se pide a los alumnos leer atentamente y **tener en cuenta** los criterios de corrección informados [en el campus](https://campusgrado.fi.uba.ar/mod/page/view.php?id=73393).

---

# Resolución

## Ejercicio 1: Docker Compose Generator

### Ejecución

Para ejecutar el ejercicio 1, primero hay que asegurarse de darle permisos de ejecución al script:

```bash
chmod +x generar-compose.sh
```

Luego, utilizar el siguiente comando desde la raíz del proyecto:

```bash
./generar-compose.sh <archivo_salida> <cantidad_clientes>
```

**Ejemplo:**

```bash
chmod +x generar-compose.sh
./generar-compose.sh docker-compose-test.yaml 5
```

Este comando generará un archivo Docker Compose con la cantidad especificada de clientes (en el ejemplo, 5 clientes: client1, client2, client3, client4, client5).

### Solución Implementada

La solución se basa en un enfoque de **parametrización de servicios** a partir de un archivo Docker Compose base existente (`docker-compose-dev.yaml`).

**Arquitectura de la solución:**

- **Script principal (`generar-compose.sh`)**: Script bash que valida parámetros y delega la generación a Python
- **Generador Python (`scripts/generate_compose.py`)**: Script que lee el archivo base, elimina el servicio del cliente preexistente y genera dinámicamente N clientes

**Aspectos destacados:**

1. **Reutilización del compose base**: En lugar de regenerar todo el archivo desde cero, se toma como base el `docker-compose-dev.yaml` existente, preservando la configuración del servidor y otros servicios
2. **Generación dinámica**: Los clientes se crean en un bucle con nombres secuenciales (`client1`, `client2`, etc.)
3. **Configuración consistente**: Cada cliente generado mantiene las mismas propiedades (imagen, networks, dependencias) pero con identificadores únicos

### Evolución del Diseño

En este ejercicio inicial se optó por **parametrizar servicios existentes** en lugar de generar un compose completo desde cero.
En ejercicios posteriores, esta estrategia evolucionó hacia la generación completa del archivo Docker Compose dentro del mismo script.

## Ejercicio 2: Configuración Externa con Volúmenes

### Ejecución

Para ejecutar el ejercicio 2, utilizar los comandos estándar de Docker Compose como indica el enunciado.

### Solución Implementada

El ejercicio se centra en **externalizar la configuración** de las aplicaciones para evitar reconstrucciones innecesarias de las imágenes Docker.

#### Problema Original

Inicialmente, los archivos de configuración (`config.yaml` y `config.ini`) estaban copiados dentro de las imágenes Docker durante el build. Esto implicaba que **cualquier cambio en la configuración requiera reconstruir completamente la imagen**, proceso que es:

- **Lento**: Requiere recompilar y reempaquetar toda la aplicación
- **Ineficiente**: Desperdicia tiempo y recursos computacionales
- **Poco práctico**: Para simples cambios de configuración

#### Solución Implementada: Docker Volumes

**1. Eliminación de archivos de configuración de los Dockerfiles:**

- Se removieron las líneas `COPY config.yaml` y `COPY config.ini` de los Dockerfiles

**2. Montaje de volúmenes en Docker Compose:**

```yaml
# Servidor Python
volumes:
  - ./server/config.ini:/config.ini

# Cliente Go
volumes:
  - ./client/config.yaml:/config.yaml
```

**3. Eliminación de variables de entorno conflictivas:**

- Se removieron variables de entorno del Docker Compose que tenían **precedencia sobre los archivos de configuración**
- Esto asegura que la configuración se lea exclusivamente desde los archivos montados

## Ejercicio 3: Validación de Echo Server con Docker Networks

### Ejecución

```bash
# Iniciar el sistema
make docker-compose-up

# Dar permisos de ejecución al script
chmod +x validar-echo-server.sh

# Ejecutar la validación
./validar-echo-server.sh
```

**Salida esperada:**

```bash
action: test_echo_server | result: success
```

Si el servidor no está funcionando correctamente:

```bash
action: test_echo_server | result: fail
```

### Solución Implementada

El ejercicio implementa un **sistema de health check** para verificar que el echo server funciona correctamente, utilizando Docker networks para la comunicación interna.

#### Desafíos del Enunciado

1. **No instalar netcat en el host**: El comando `nc` no debe instalarse en la máquina local
2. **No exponer puertos**: La comunicación debe ser interna, sin mapear puertos del servidor al host
3. **Usar netcat para testing**: Herramienta estándar para testing de conectividad TCP

#### Flujo de Validación

1. **Creación del container temporal**: Docker crea un container basado en `busybox`
2. **Conexión a la red**: Se une a `tp0_testing_net` (misma red del servidor)
3. **Envío del mensaje**: `echo 'Hello Echo Server!' | nc server 12345`
4. **Resolución DNS interna**: Utilizamos `server` como hostname (Docker DNS interno)
5. **Recepción y validación**: El script verifica que el mensaje retornado sea idéntico
6. **Limpieza automática**: El container se destruye (`--rm`)

## Ejercicio 4: Graceful Shutdown con Señales del Sistema

### Ejecución

Para ejecutar el ejercicio 4 y probar el graceful shutdown:

```bash
# Iniciar el sistema
make docker-compose-up

# En otra terminal, probar el graceful shutdown
make docker-compose-down
```

**Observar los logs de cierre:**

```bash
# Ver logs durante el shutdown
make docker-compose-logs
```

**Salida esperada en los logs:**

```
server   | action: shutdown | result: in_progress | msg: received SIGTERM
server   | action: shutdown | result: success | msg: server socket closed
client1  | action: shutdown | result: in_progress | client_id: 1 | msg: received SIGTERM
client1  | action: shutdown | result: success | client_id: 1 | msg: socket closed
```

### Solución Implementada

El ejercicio implementa **graceful shutdown** mediante el manejo de señales del sistema operativo, asegurando que todos los file descriptors se cierren correctamente antes de terminar la aplicación.

#### Problema del Shutdown Abrupto

Sin graceful shutdown, cuando Docker detiene containers:

- **Conexiones TCP** quedan abiertas en estado inconsistente
- **Archivos** pueden quedar con buffers sin escribir
- **Threads** terminan abruptamente sin cleanup
- **Recursos del SO** pueden quedar sin liberar

#### Arquitectura de la Solución

**Principio de diseño**: Hacer lo mínimo e indispensable en el signal handler, usar flags para coordinar el shutdown en el flujo principal. Una vez que el flag shutdown_requested == true debe comenzar el cierre/limpieza del programa antes que finalice el thread principal. 

## Servidor

Cuando el servidor recibe la signal **SIGTERM** se ejecuta este *handler*.  

```python
def _signal_handler(self, signum, frame):
        """Handle SIGTERM signal for graceful shutdown"""
        logging.info("action: shutdown | result: in_progress | msg: received SIGTERM")
        self._shutdown_requested = True

        # Close server socket to stop accepting new connections
        try:
            self._server_socket.close()
            self._cleanup_connections()
            logging.info(
                "action: shutdown | result: success | msg: server socket closed"
            )
        except Exception as e:
            logging.error(
                f"action: shutdown | result: fail | msg: error closing server socket | error: {e}"
            )
```
```python
def _cleanup_connections(self):
        """Close active connection if exists"""

        # Close the currently active client connection, if it exists
        if self._client_socket:
            try:
                self._client_socket.shutdown(socket.SHUT_RDWR)
                self._client_socket.close()
                logging.info(
                    "action: shutdown | result: success | msg: client connection closed"
                )
            except Exception as e:
                logging.error(
                    f"action: shutdown | result: fail | msg: error closing client connection | error: {e}"
                )
            self._client_socket = None
```

Podemos observar que el flag `self._shutdown_requested` se marca en `true`, lo que permite luego al loop principal del servidor finalizar, ya que el socket principal que acepta conexiones se ha cerrado.  

Por otro lado, se le notifica al cliente que la conexión ha terminado `(self._client_socket.shutdown(socket.SHUT_RDWR))`, para que luego el cliente pueda manejar esta signal del otro lado y terminar correctamente, ya que no se aceptará más el envío de mensajes.  

Más adelante (en particular en el **ejercicio 8**), el manejo del *shutdown* será modificado levemente para poder notificarles a los *threads* de la signal **SIGTERM** y que puedan finalizar correctamente.  

A su vez, también se tendrá en cuenta el cierre del archivo CSV, ya que ante una signal **SIGTERM**, también se debe garantizar el cierre correcto de este *file descriptor*. 

## Cliente

En el siguiente fragmento de código, podemos visualizar cómo, del lado del cliente, hacemos un manejo similar en cuanto a la signal **SIGTERM**.  

Cerramos el socket principal, el flag `client.shutdownRequested` se cambia a `true` y, de esta forma, el programa será notificado de la signal y podrá salir correctamente del programa.
```Go
func NewClient(config ClientConfig) *Client {
	client := &Client{config: config}

	// Setup signal handler for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-sigChan
		log.Infof("action: shutdown | result: in_progress | client_id: %v | msg: received SIGTERM", client.config.ID)
		client.shutdownRequested = true

		if client.conn != nil {
			client.conn.Close()
			log.Infof("action: shutdown | result: success | client_id: %v | msg: socket closed", client.config.ID)
		}
	}()

	return client
}
```

A su vez, en caso de que sea el **servidor** quien realice el *shutdown*, tendremos la siguiente verificación en el socket, que nos indicará si se ha cerrado el otro extremo de la conexión.  

```Go
if err == io.EOF {
    log.Infof("action: receive_message | result: server_shutdown | client_id: %v | msg: server closed connection",
        c.config.ID,
    )
}
```





## Ejercicio 5: Protocolo de Comunicación Personalizado

### Ejecución

Para ejecutar el ejercicio 5 con el nuevo sistema de apuestas:

```bash
# Iniciar el sistema
make docker-compose-up

# Ver logs para observar las apuestas
make docker-compose-logs
```

**Salida esperada en los logs:**

```
client1  | action: apuesta_enviada | result: success | dni: 30904465 | numero: 7574
server   | action: apuesta_almacenada | result: success | dni: 30904465 | numero: 7574
```

### Solución Implementada

El ejercicio transforma el sistema de echo server en un **sistema de apuestas de quiniela**, implementando un protocolo de comunicación completo entre clientes (agencias) y servidor (central de lotería).

#### Caso de Uso: Lotería Nacional

**Clientes (Agencias de Quiniela)**:

- Existen **5 agencias** numeradas del 1 al 5
- Reciben datos de apuestas via **variables de entorno**
- Envían apuestas al servidor central
- Confirman registro exitoso

**Servidor (Central de Lotería Nacional)**:

- Recibe apuestas de todas las agencias
- Almacena usando función `store_bet()` provista por la cátedra
- Responde confirmaciones a clientes

#### Datos de Apuesta

Cada apuesta contiene los siguientes campos:

- **NOMBRE**: Nombre del apostador
- **APELLIDO**: Apellido del apostador
- **DOCUMENTO**: DNI del apostador
- **NACIMIENTO**: Fecha de nacimiento (formato: YYYY-MM-DD)
- **NUMERO**: Número apostado

### Protocolo de Comunicación Personalizado

#### Arquitectura del Protocolo

El protocolo implementado utiliza un esquema **length-prefixed** con mensajes serializados en formato custom.

**Estructura del mensaje:**

```
[4 bytes LENGTH][MESSAGE_TYPE][PAYLOAD]
```

- **LENGTH**: Tamaño del mensaje en bytes (big-endian, 4 bytes)
- **MESSAGE_TYPE**: Tipo de mensaje (1 byte)
- **PAYLOAD**: Datos del mensaje

#### Tipos de Mensaje

```python
MESSAGE_TYPE_BET = 1         # Apuesta individual
MESSAGE_TYPE_BATCH = 2       # Batch de apuestas
MESSAGE_TYPE_RESPONSE = 3    # Respuesta del servidor
MESSAGE_TYPE_GET_WINNERS = 4 # Consulta de ganadores
```

#### Mensajes Implementados (para este ejercicio)

**1. BetMessage (TYPE = 1)**

```
Formato: 1|NOMBRE|APELLIDO|DOCUMENTO|NACIMIENTO|NUMERO
Ejemplo: 1|Santiago Lionel|Lorca|30904465|1999-03-17|7574
```

**2. ResponseMessage (TYPE = 3)**

```
Formato: 3|SUCCESS|ERROR|WINNER_COUNT|WINNER1|WINNER2|...
Ejemplo: 3|1||0|  (éxito sin ganadores)
```

### Prevención de Short Read/Write

#### Problema de Short Operations

En comunicación TCP, las operaciones `send()` y `recv()` **no garantizan transferir todos los bytes solicitados** en una sola llamada. Esto puede causar:

- **Short Write**: `send()` transmite menos bytes de los solicitados
- **Short Read**: `recv()` recibe menos bytes de los esperados

#### Solución Implementada

**Read Exacto en Python:**

```python
def _read_exact(sock, n):
    """Read exactly n bytes from socket"""
    chunks = []
    bytes_read = 0
    while bytes_read < n:
        chunk = sock.recv(n - bytes_read)
        if not chunk:  # Connection closed
            raise RuntimeError("Socket connection broken")
        chunks.append(chunk)
        bytes_read += len(chunk)
    return b"".join(chunks)
```

**Read Exacto en Go:**

```go
func readExact(conn net.Conn, buf []byte) (int, error) {
    totalRead := 0
    for totalRead < len(buf) {
        n, err := conn.Read(buf[totalRead:])
        if err != nil {
            return totalRead, err
        }
        totalRead += n
    }
    return totalRead, nil
}
```


### Ejemplo de Serialización y Deserialización

Para ilustrar el proceso completo de comunicación, veamos un ejemplo paso a paso del envío de una apuesta:

#### **Lado Cliente (Go) - Serialización:**

```go
// 1. Crear el mensaje de apuesta
bet := &BetMessage{
    Type:       MessageTypeBet,
    Nombre:     "Santiago Lionel",
    Apellido:   "Lorca",
    Documento:  "30904465",
    Nacimiento: "1999-03-17",
    Numero:     7574,
}

// 2. Serializar a formato custom
func SerializeMessage(message *BetMessage) ([]byte, error) {
    data := []byte{byte(MessageTypeBet)}  // Tipo: 1 byte
    content := fmt.Sprintf("%s|%s|%s|%s|%d",
        bet.Nombre, bet.Apellido, bet.Documento, bet.Nacimiento, bet.Numero)
    data = append(data, []byte(content)...)
    return data, nil
}
// Resultado: [1]Santiago Lionel|Lorca|30904465|1999-03-17|7574

// 3. Enviar con length-prefix
func SendMessage(conn net.Conn, message *BetMessage) error {
    data, _ := SerializeMessage(message)

    // Calcular y enviar longitud (4 bytes big-endian)
    length := uint32(len(data))  // length = 51 bytes
    lengthBytes := make([]byte, 4)
    binary.BigEndian.PutUint32(lengthBytes, length)
    conn.Write(lengthBytes)      // [0, 0, 0, 51]

    // Enviar datos del mensaje
    conn.Write(data)             // [1]Santiago Lionel|Lorca|30904465|1999-03-17|7574
    return nil
}
```

#### **Lado Servidor (Python) - Deserialización:**

```python
# 1. Leer length-prefix
def read_packet_from(client_socket):
    # Leer exactamente 4 bytes para la longitud
    length_data = _read_exact(client_socket, 4)  # [0, 0, 0, 51]
    length = int.from_bytes(length_data, byteorder='big')  # length = 51

    # Leer exactamente 'length' bytes de datos
    data = _read_exact(client_socket, length)    # [1]Santiago Lionel|Lorca|...

    # 2. Parsear según tipo de mensaje
    msg_type = data[0]  # msg_type = 1 (MESSAGE_TYPE_BET)

    if msg_type == MESSAGE_TYPE_BET:
        return BetMessage.from_data(data)

# 3. Deserializar el contenido
class BetMessage:
    @classmethod
    def from_data(cls, data):
        content = data[1:].decode('utf-8')  # "Santiago Lionel|Lorca|30904465|1999-03-17|7574"
        parts = content.split('|')          # ["Santiago Lionel", "Lorca", "30904465", ...]

        return cls(
            type=MESSAGE_TYPE_BET,
            nombre=parts[0],      # "Santiago Lionel"
            apellido=parts[1],    # "Lorca"
            documento=parts[2],   # "30904465"
            nacimiento=parts[3],  # "1999-03-17"
            numero=int(parts[4])  # 7574
        )
```

#### **Protocolo Extensible: BatchMessage** (utilizado en ejercicios posteriores)

**Lado Cliente (Go) - Serialización de Batch:**

```go
// 1. Crear un batch con múltiples apuestas
batch := &BatchMessage{
    Type:   MessageTypeBatch,
    Agency: 1,
    EOF:    false,
    Bets: []BetMessage{
        {Nombre: "Juan", Apellido: "Perez", Documento: "12345678", Nacimiento: "1990-01-01", Numero: 1234},
        {Nombre: "Maria", Apellido: "Garcia", Documento: "87654321", Nacimiento: "1985-05-15", Numero: 5678},
        {Nombre: "Carlos", Apellido: "Lopez", Documento: "11223344", Nacimiento: "1992-12-30", Numero: 9012},
    },
}

// 2. Serializar BatchMessage
func SerializeMessage(message *BatchMessage) ([]byte, error) {
    data := []byte{byte(MessageTypeBatch)}  // Tipo: 2

    // Formato: AGENCY|EOF|COUNT|BET1|BET2|BET3|...
    content := fmt.Sprintf("%d|%d|%d", msg.Agency, boolToInt(msg.EOF), len(msg.Bets))
    for _, bet := range msg.Bets {
        content += fmt.Sprintf("|%s|%s|%s|%s|%d",
            bet.Nombre, bet.Apellido, bet.Documento, bet.Nacimiento, bet.Numero)
    }
    data = append(data, []byte(content)...)
    return data, nil
}

// Resultado: [2]1|0|3|Juan|Perez|12345678|1990-01-01|1234|Maria|Garcia|87654321|1985-05-15|5678|Carlos|Lopez|11223344|1992-12-30|9012
```

**Lado Servidor (Python) - Deserialización de Batch:**

```python
# 1. Leer y parsear BatchMessage
class BatchMessage:
    @classmethod
    def from_data(cls, data):
        content = data[1:].decode('utf-8')
        parts = content.split('|')

        agency = int(parts[0])      # 1
        eof = parts[1] == '1'       # False (0)
        bet_count = int(parts[2])   # 3

        # Parsear cada apuesta (5 campos por apuesta)
        bets = []
        start_idx = 3
        for i in range(bet_count):
            if idx + 4 >= len(parts):
                break
            bet = BetMessage(
                nombre=parts[start_idx],      # Juan, Maria, Carlos
                apellido=parts[start_idx + 1], # Perez, Garcia, Lopez
                documento=parts[start_idx + 2], # 12345678, 87654321, 11223344
                nacimiento=parts[start_idx + 3], # 1990-01-01, 1985-05-15, 1992-12-30
                numero=int(parts[start_idx + 4]) # 1234, 5678, 9012
            )
            bets.append(bet)
            idx += 5

        return cls(type=MESSAGE_TYPE_BATCH, agency=agency, eof=eof, bets=bets)

```

### Conceptos de Comunicación

**Length-Prefixed Protocol**: Patrón estándar que resuelve el problema de **message framing** en streams TCP. Sin delimitadores de mensaje, el receptor no sabe dónde termina un mensaje y empieza el siguiente.

**Custom Serialization**: Formato optimizado para el dominio específico vs protocolos genéricos como JSON/XML. Menor overhead y mayor control sobre el formato.

## Ejercicio 6: Procesamiento por Batches (Chunks)

### Solución Implementada

El ejercicio evoluciona el sistema para **procesamiento por batches**, donde los clientes envían múltiples apuestas agrupadas en lugar de apuestas individuales, optimizando el rendimiento de red y procesamiento.

### Ingesta de Datos desde CSV

**Fuente de datos**: Archivos `.data/agency-{N}.csv` (donde N = 1-5)

**Convención de archivos:**

- `client1` lee `.data/agency-1.csv`
- `client2` lee `.data/agency-2.csv`
- ... y así sucesivamente

#### Implementación del BatchMessage

**Estructura del mensaje:**

```go
type BatchMessage struct {
    Type   MessageType      // MESSAGE_TYPE_BATCH (2)
    Agency int             // Número de agencia (1-5)
    Bets   []BetMessage    // Array de apuestas
}
```

#### Lógica de Batching

**Criterios para enviar un batch:**

1. **Límite de cantidad**: Configurado en `config.yaml` (`batch.maxAmount`)
2. **Límite de tamaño**: Máximo 8KB por paquete (constante `MAX_BATCH_SIZE_BYTES`)

```go
const MAX_BATCH_SIZE_BYTES = 8 * 1024 // 8kB limite
```

El sistema utiliza una lógica **OR** entre ambos criterios - el batch se envía cuando **cualquiera** de las dos condiciones se cumple (ver codigo):

**Escenario 1 - Límite de tamaño alcanzado:**

```
max_amount = 10 bets
test_batch = 8 bets
batch_size = 8.1 KB -> EXCEDE
current_batch = 7 bets → ENVIAMOS
```

**Escenario 2 - Límite de cantidad alcanzado:**

```
max_amount = 10 bets
test_batch = 11 bets -> EXCEDE
batch_size = 6.5 KB → NO EXCEDE
current_batch = 10 bets -> ENVIAMOS
```

**Escenario 3 - Continuar acumulando:**

```
max_amount = 10 bets
test_batch = 8 bets
batch_size = 5.2 KB → CONTINUAR (ambos límites permiten más)
current_batch = 7 bets -> sumamos un bet mas -> 7 + 1 = 8 bets
```

## Ejercicio 7: Sorteo y Consulta de Ganadores

### Solución Implementada

El ejercicio implementa el **flujo completo de sorteo**, donde los clientes notifican el fin de apuestas, el servidor coordina el sorteo tras recibir todas las notificaciones, y los clientes (agencias) consultan sus ganadores específicos una vez que envian sus apuestas.

### Flujo del sorteo:

1. **Envío de apuestas**: Clientes envían batches con apuestas
2. **Notificación EOF**: Último batch de cada agencia tiene `EOF=true`
3. **Coordinación del sorteo**: Servidor espera notificación de las 5 agencias
4. **Ejecución del sorteo**: Uso de funciones `load_bets()` y `has_won()` provistas por la cátedra
5. **Consulta de ganadores**: Clientes solicitan ganadores específicos de su agencia

### Notificación de Fin de Apuestas (EOF)

En lugar de crear un mensaje dedicado para EOF, se reutiliza el `BatchMessage` existente con un flag:

```go
type BatchMessage struct {
    Type   MessageType
    Agency int
    Bets   []BetMessage
    EOF    bool           // Indica si es el último batch
}
```

#### Consulta de Ganadores

**Nuevo tipo de mensaje:**

```go
type GetWinnersMessage struct {
    Type   MessageType  // = MESSAGE_TYPE_GET_WINNERS (4)
    Agency int         // Número de agencia consultante
}
```

### Archivos Importantes Modificados

#### `client/common/client.go`

- **Nuevo**: Función para consulta de ganadores después de EOF
- **Modificado**: `StartClientLoopWithCSV()` para enviar EOF en último batch
- **Lógica**: Flujo de envío → EOF → consulta de ganadores. (Se implemento una politica de retries por si el sorteo aun no se ha realizado)

#### `server/common/server.py`

- **Nuevo**: Set `finished_agencies` para tracking de EOF de las agencias
- **Nuevo**: `perform_lottery()` que dentro utiliza funciones provistas por la cátedra
- **Validación**: Consultas de los ganadores antes de tiempo son rechazadas

## Ejercicio 8: Conexiones Persistentes y Concurrencia con Multithreading

### Solución Implementada

El ejercicio transforma el servidor de **conexiones por mensaje** a **conexiones persistentes**, implementando procesamiento concurrente mediante multithreading para manejar múltiples clientes simultáneamente.

### Cambio Fundamental: Conexiones Persistentes

**Antes (Ejercicios 1-7)**:

- Conexión nueva para cada batch
- Cliente: conectar → enviar batch → desconectar

**Ahora (Ejercicio 8)**:

- Una conexión por cliente durante toda la sesión
- Cliente: conectar → enviar batches + consultar ganadores (politica de retries) → desconectar
- Reutilización de conexión TCP

### Refactorización de la Arquitectura del Servidor

#### Modularización en Tres Componentes

**1. `Server.py` - Lógica de Negocio**

- Core business logic: sorteo, validación, almacenamiento
- Thread-safe data structures con locks
- Callbacks para delegación a otros módulos

**2. `Listener.py` - Aceptación de Conexiones**

- Acepta nuevas conexiones en loop principal
- Crea un `ClientHandler` thread por cada cliente
- Maneja shutdown graceful coordinado

**3. `ClientHandler.py` - Thread por Cliente**

- Hereda de `threading.Thread`
- Maneja comunicación persistente con un cliente específico
- Ejecuta callbacks del server para delegación de lógica

#### Patrón de Callbacks

```python
# En Server.py - creación de callbacks
server_callbacks = {
    "handle_batch_message": self._handle_batch_message,
    "handle_get_winners_message": self._handle_get_winners_message,
}

# En ClientHandler.py - ejecución de callbacks
self.server_callbacks["handle_batch_message"](message)
success = self.server_callbacks["handle_get_winners_message"](message, socket)
```

**¿Por qué callbacks en lugar de incluir lógica en ClientHandler?**

- **Separación de responsabilidades**: ClientHandler = comunicación, Server = business logic

### ¿Por qué Multithreading?

El factor más importante de por qué decidí utilizarlo fue por la familiaridad con la herramienta en Python. En un contexto en donde el cómputo sea intensivo y hacer uso del multicore sea necesario, se utilizaría multi-processing. Aunque hay que destacar que crear procesos es más pesado que crear threads. Por lo que hay que tener en cuenta el scope del problema para utilizar una herramienta u otra.

Por otro lado, mientras actualmente el cliente hace un estilo de polling incremental para consultar los ganadores, otra opción superadora para este caso hubiese sido utilizar barriers. El cliente, por su lado, solo tendría que ejecutar una única request, y el servidor, sabiendo si se ha sorteado o no la lotería, dejará que la request de la agencia (cliente) avance o no. Si la lotería se ha realizado, dejará que la request avance y le devolverá los ganadores; si no, deberá esperar hasta que se realice el sorteo. De esta forma, se minimiza el envío de requests en la red y se administran mejor los recursos de los servicios.

En conclusión, si bien la última opción puede resultar superadora, considero que, para el scope de este trabajo práctico, no es necesaria su implementación

### Graceful Shutdown Mejorado con Cleanup Callback

#### Problema del Tracking Manual de Threads

En versiones anteriores, el `Listener` tenía que mantener manualmente una lista de handlers activos/inactivos para el shutdown graceful. Es decir, si un thread finalizaba, la lista seguia manteniendo a ese thread ya finalizado.

#### Solución: Cleanup Callback

Se implementó un **mecanismo de callback automático** donde los handlers se auto-remueven de la lista cuando terminan:

```python
def run(self):
    try:
        self._handle_client_communication()
    finally:
        self._cleanup_connection()
        if self.cleanup_callback:
            try:
                self.cleanup_callback(self)
            except Exception as e:
                self._log_action("cleanup_callback", "fail", level=logging.ERROR, error=e)
```

#### Implementación del Listener con Callback

```python
def _remove_handler(self, handler):
    """Remove a finished handler from the active handlers set"""
    try:
        with self._handlers_lock:
            self._active_handlers.discard(handler)
        logging.debug(f"action: remove_handler | result: success | ip: {handler.client_address[0]}")
    except Exception as e:
        logging.error(f"action: remove_handler | result: fail | error: {e}")
```

#### Graceful Shutdown con SIGTERM

Cuando el `Listener` recibe una señal `SIGTERM`, ejecuta el siguiente proceso:

1. **Cierre del socket servidor**: Para de aceptar nuevas conexiones
2. **Notificación a handlers activos**: Cada handler recibe `request_shutdown()`
3. **Cleanup de conexiones**: Cada handler ejecuta su cleanup individual

```python
def request_shutdown(self):
    """Request graceful shutdown of this handler"""
    self._shutdown_requested = True
    try:
        self.client_socket.shutdown(socket.SHUT_RDWR)
        self.client_socket.close()
    except Exception as e:
        self._log_action("close_connection", "fail", level=logging.ERROR, error=e)
```

#### Ventajas del Mecanismo de Callbacks

1. **Automático**: Los handlers se auto-remueven sin intervención manual del Listener
2. **Thread-safe**: Uso de locks para operaciones concurrentes sobre `_active_handlers`
3. **Separación de responsabilidades**: Cada handler maneja su propio cleanup