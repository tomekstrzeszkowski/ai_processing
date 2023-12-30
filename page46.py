import numpy as np
import pandas as pd
import matplotlib.pyplot as plt
from dataclasses import dataclass


def get_dataframe():
	return pd.read_csv("./iris/iris.data", header=None)


def get_y_values(data_frame):
	y = data_frame.iloc[:100, 4].values
	return np.where(y == "Iris-setosa", -1, 1)	


def get_x_values(data_frame):
	return data_frame.iloc[:100, [0, 2]].values


@dataclass
class Perceptron:
	eta: float = 0.1
	n_iter: int = 10
	random_state: int = 1

	def fit(self, X, y):
		rgen = np.random.RandomState(self.random_state)
		self.w = rgen.normal(loc=0.0, scale=.01, size=1+X.shape[1])
		self.errors = []

		for _ in range(self.n_iter):
			errors = 0
			for xi, target in zip(X, y):
				update = self.eta * (target - self.predict(xi))
				self.w[1:] += update * xi
				self.w[0] += update
				errors += int(update != 0.0)
			self.errors.append(errors)
		return self

	def net_input(self, X):
		return np.dot(X, self.w[1:] + self.w[0])

	def predict(self, X):
		return np.where(self.net_input(X) >= 0.0, 1, -1)


if __name__ == "__main__":
	perceptron = Perceptron()
	data_frame = get_dataframe()
	perceptron.fit(get_x_values(data_frame), get_y_values(data_frame))

	plt.plot(range(1, len(perceptron.errors) + 1), perceptron.errors, marker="o")
	plt.xlabel("Epochs")
	plt.ylabel("Uptade counter")
	plt.show()