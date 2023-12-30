import numpy as np
import pandas as pd
import matplotlib.pyplot as plt

from page46 import Perceptron, get_dataframe, get_x_values, get_y_values


class AdalineGD(Perceptron):
	def fit(self, X, y):
		rgen = np.random.RandomState(self.random_state)
		self.w = rgen.normal(loc=0.0, scale=0.01, size=1+X.shape[1])
		self.cost = []
		for i in range(self.n_iter):
			net_input = self.net_input(X)
			output = self.activation(net_input)
			errors = (y - output)
			self.w[1:] += self.eta * X.T.dot(errors)
			self.w[0] += self.eta * errors.sum()
			cost = (errors ** 2).sum() / 2.
			self.cost.append(cost)
		return self

	def net_input(self, X):
		return np.dot(X, self.w[1:]) + self.w[0]

	def activation(self, X):
		return X

	def predict(self, X):
		active = self.activation(self.net_input(X))
		return np.where(active >= 0.0, 1, -1)


if __name__ == "__main__":
	data_frame = get_dataframe()
	X = get_x_values(data_frame)
	y = get_y_values(data_frame)
	fig, ax = plt.subplots(nrows=1, ncols=2, figsize=(10, 4))
	
	ada1 = AdalineGD(n_iter=10, eta=0.01).fit(X, y)
	ada2 = AdalineGD(n_iter=10, eta=0.0001).fit(X, y)

	ax[0].plot(range(1, len(ada1.cost) + 1), np.log10(ada1.cost), marker="o")
	ax[0].set_xlabel("Epochs")
	ax[0].set_ylabel("Log (error sum)")
	ax[0].set_title("Adaline 0.01")

	ax[1].plot(range(1, len(ada2.cost) + 1), np.log10(ada2.cost), marker="o")
	ax[1].set_xlabel("Epochs")
	ax[1].set_ylabel("Log (error sum)")
	ax[1].set_title("Adaline 0.0001")
	
	plt.show()