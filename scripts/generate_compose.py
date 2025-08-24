import sys
import yaml


def generate_compose(output_file, n_clients):
    # 1. Leer el archivo base
    with open("docker-compose-dev.yaml") as f:
        compose = yaml.safe_load(f)

    # 2. Eliminar el client1 que ya viene fijo 
    compose["services"].pop("client", None)

    # 3. Agregar N clientes dinámicamente
    for i in range(1, n_clients + 1):
        compose["services"][f"client{i}"] = {
            "container_name": f"client{i}",
            "image": "client:latest",
            "entrypoint": "/client",
            "environment": [f"CLI_ID={i}", "CLI_LOG_LEVEL=DEBUG"],
            "networks": ["testing_net"],
            "depends_on": ["server"],
        }

    # 4. Guardar el nuevo compose en archivo de salida
    with open(output_file, "w") as f:
        yaml.dump(compose, f, sort_keys=False)


if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Usage: python3 generate_compose.py <output_file> <n_clients>")
        sys.exit(1)

    output_file = sys.argv[1]
    n_clients = int(sys.argv[2])
    generate_compose(output_file, n_clients)
