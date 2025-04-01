from locust import HttpUser, task
import random


class HotelWebsiteUser(HttpUser):
    def get_user(self):
        id = random.randint(0, 500)
        user_name = f"Cornell_{id}"
        pass_word = str(id) * 10
        return user_name, pass_word

    def search_hotel(self):
        in_date = random.randint(9, 23)
        out_date = random.randint(in_date + 1, 24)

        in_date_str = f"2015-04-{in_date:02d}"
        out_date_str = f"2015-04-{out_date:02d}"

        lat = 38.0235 + (random.randint(0, 481) - 240.5) / 1000.0
        lon = -122.095 + (random.randint(0, 325) - 157.0) / 1000.0

        params = {
            "inDate": in_date_str,
            "outDate": out_date_str,
            "lat": lat,
            "lon": lon,
        }

        self.client.get("/hotels", params=params)

    def recommend(self):
        coin = random.random()
        if coin < 0.33:
            req_param = "dis"
        elif coin < 0.66:
            req_param = "rate"
        else:
            req_param = "price"

        lat = 38.0235 + (random.randint(0, 481) - 240.5) / 1000.0
        lon = -122.095 + (random.randint(0, 325) - 157.0) / 1000.0

        params = {"require": req_param, "lat": lat, "lon": lon}

        self.client.get("/recommendations", params=params)

    def reserve(self):
        in_date = random.randint(9, 23)
        out_date = in_date + random.randint(1, 5)

        in_date_str = f"2015-04-{in_date:02d}"
        out_date_str = f"2015-04-{out_date:02d}"

        hotel_id = str(random.randint(1, 80))
        user_id, password = self.get_user()
        cust_name = user_id
        num_room = "1"

        params = {
            "inDate": in_date_str,
            "outDate": out_date_str,
            "hotelId": hotel_id,
            "customerName": cust_name,
            "username": user_id,
            "password": password,
            "number": num_room,
        }

        self.client.post("/reservation", params=params)

    def user_login(self):
        user_name, password = self.get_user()
        params = {"username": user_name, "password": password}
        self.client.post("/user", params=params)

    @task
    def mixed_workload(self):
        coin = random.random()
        search_ratio = 0.6
        recommend_ratio = 0.39
        user_ratio = 0.005

        if coin < search_ratio:
            self.search_hotel()
        elif coin < search_ratio + recommend_ratio:
            self.recommend()
        elif coin < search_ratio + recommend_ratio + user_ratio:
            self.user_login()
        else:
            self.reserve()
