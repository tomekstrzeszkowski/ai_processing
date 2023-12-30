import numpy as np
import pandas as pd
import matplotlib.pyplot as plt
from numpy.random import seed
from page56 import AdalineGD
from dataclasses import dataclass
from mlxtend.plotting import plot_decision_regions

from page46 import get_dataframe, get_x_values, get_y_values


@dataclass
class AdalineSGD(AdalineGD):
	eta: float = 0.01
	w_initialized: bool = False
	shuffle: bool = True

	def fit(self, X, y):
		self.initialize_weights(X.shape[1])
		self.cost = []
		for i in range(self.n_iter):
			if self.shuffle:
				X, y = self._shuffle(X, y)
			cost = []
			for xi, target in zip(X,y):
				cost.append(self.update_weights(xi, target))
			avg_cost = sum(cost) / len(y)
			self.cost.append(avg_cost)
		return self

	def initialize_weights(self, m):
		self.rgen = np.random.RandomState(self.random_state)
		self.w = self.rgen.normal(loc=0.0, scale=0.01, size=1+m)
		self.w_initialized = True

	def _shuffle(self, X, y):
		r = self.rgen.permutation(len(y))
		return X[r], y[r]

	def update_weights(self, xi, target):
		output = self.activation(self.net_input(xi))
		error = target - output
		self.w[1:] += self.eta * xi.dot(error)
		self.w[0] += self.eta * error
		return .5 * error ** 2


if __name__ == "__main__":
	data_frame = get_dataframe()
	X = get_x_values(data_frame)
	y = get_y_values(data_frame)
	ada = AdalineSGD()
	ada.fit(X, y)