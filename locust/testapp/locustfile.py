from locust import HttpUser, task
import random
import string


class TestappUser(HttpUser):
    def __init__(self):
        self.characters = string.ascii_letters + string.digits

    @task
    def workload(self):
        self.client.get("/")

        data = {
            "data": "".join(
                random.choice(self.characters) for _ in range(random.randint(5, 20))
            )
        }
        self.client.get("/noop", data=data)
