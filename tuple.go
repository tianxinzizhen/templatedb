package templatedb

// ------- T
type T[T0 any] struct {
	T0 T0
}

func NewT[T0 any](t0 T0) T[T0] {
	return T[T0]{t0}
}

func (t T[T0]) Values() T0 {
	return t.T0
}

// ------- T2
type T2[T0 any, T1 any] struct {
	T0 T0
	T1 T1
}

func NewT2[T0 any, T1 any](t0 T0, t1 T1) T2[T0, T1] {
	return T2[T0, T1]{t0, t1}
}

func (t T2[T0, T1]) Values() (T0, T1) {
	return t.T0, t.T1
}

// ------- T3
type T3[T0 any, T1 any, T2 any] struct {
	T0 T0
	T1 T1
	T2 T2
}

func NewT3[T0 any, T1 any, T2 any](
	t0 T0, t1 T1, t2 T2) T3[
	T0, T1, T2] {
	return T3[T0, T1, T2]{t0, t1, t2}
}

func (t T3[T0, T1, T2]) Values() (T0, T1, T2) {
	return t.T0, t.T1, t.T2
}

// ------- T4
type T4[T0 any, T1 any, T2 any, T3 any] struct {
	T0 T0
	T1 T1
	T2 T2
	T3 T3
}

func NewT4[T0 any, T1 any, T2 any, T3 any](
	t0 T0, t1 T1, t2 T2, t3 T3) T4[
	T0, T1, T2, T3] {
	return T4[T0, T1, T2, T3]{t0, t1, t2, t3}
}

func (t T4[T0, T1, T2, T3]) Values() (
	T0, T1, T2, T3) {
	return t.T0, t.T1, t.T2, t.T3
}

// ------- T5
type T5[T0 any, T1 any, T2 any, T3 any, T4 any] struct {
	T0 T0
	T1 T1
	T2 T2
	T3 T3
	T4 T4
}

func NewT5[T0 any, T1 any, T2 any, T3 any, T4 any](
	t0 T0, t1 T1, t2 T2, t3 T3, t4 T4) T5[
	T0, T1, T2, T3, T4] {
	return T5[T0, T1, T2, T3, T4]{t0, t1, t2, t3, t4}
}

func (t T5[T0, T1, T2, T3, T4]) Values() (
	T0, T1, T2, T3, T4) {
	return t.T0, t.T1, t.T2, t.T3, t.T4
}

// ------- T6
type T6[T0 any, T1 any, T2 any, T3 any, T4 any, T5 any] struct {
	T0 T0
	T1 T1
	T2 T2
	T3 T3
	T4 T4
	T5 T5
}

func NewT6[T0 any, T1 any, T2 any, T3 any, T4 any, T5 any](
	t0 T0, t1 T1, t2 T2, t3 T3, t4 T4, t5 T5) T6[
	T0, T1, T2, T3, T4, T5] {
	return T6[T0, T1, T2, T3, T4, T5]{t0, t1, t2, t3, t4, t5}
}

func (t T6[T0, T1, T2, T3, T4, T5]) Values() (
	T0, T1, T2, T3, T4, T5) {
	return t.T0, t.T1, t.T2, t.T3, t.T4, t.T5
}

// ------- T7
type T7[T0 any, T1 any, T2 any, T3 any, T4 any, T5 any, T6 any] struct {
	T0 T0
	T1 T1
	T2 T2
	T3 T3
	T4 T4
	T5 T5
	T6 T6
}

func NewT7[T0 any, T1 any, T2 any, T3 any, T4 any, T5 any, T6 any](
	t0 T0, t1 T1, t2 T2, t3 T3, t4 T4, t5 T5, t6 T6) T7[
	T0, T1, T2, T3, T4, T5, T6] {
	return T7[T0, T1, T2, T3, T4, T5, T6]{t0, t1, t2, t3, t4, t5, t6}
}

func (t T7[T0, T1, T2, T3, T4, T5, T6]) Values() (
	T0, T1, T2, T3, T4, T5, T6) {
	return t.T0, t.T1, t.T2, t.T3, t.T4, t.T5, t.T6
}

// ------- T8
type T8[T0 any, T1 any, T2 any, T3 any, T4 any, T5 any, T6 any, T7 any] struct {
	T0 T0
	T1 T1
	T2 T2
	T3 T3
	T4 T4
	T5 T5
	T6 T6
	T7 T7
}

func NewT8[T0 any, T1 any, T2 any, T3 any, T4 any, T5 any, T6 any, T7 any](
	t0 T0, t1 T1, t2 T2, t3 T3, t4 T4, t5 T5, t6 T6, t7 T7) T8[
	T0, T1, T2, T3, T4, T5, T6, T7] {
	return T8[T0, T1, T2, T3, T4, T5, T6, T7]{t0, t1, t2, t3, t4, t5, t6, t7}
}

func (t T8[T0, T1, T2, T3, T4, T5, T6, T7]) Values() (
	T0, T1, T2, T3, T4, T5, T6, T7) {
	return t.T0, t.T1, t.T2, t.T3, t.T4, t.T5, t.T6, t.T7
}

// ------- T9
type T9[T0 any, T1 any, T2 any, T3 any, T4 any, T5 any, T6 any, T7 any, T8 any] struct {
	T0 T0
	T1 T1
	T2 T2
	T3 T3
	T4 T4
	T5 T5
	T6 T6
	T7 T7
	T8 T8
}

func NewT9[T0 any, T1 any, T2 any, T3 any, T4 any, T5 any, T6 any, T7 any, T8 any](
	t0 T0, t1 T1, t2 T2, t3 T3, t4 T4, t5 T5, t6 T6, t7 T7, t8 T8) T9[
	T0, T1, T2, T3, T4, T5, T6, T7, T8] {
	return T9[T0, T1, T2, T3, T4, T5, T6, T7, T8]{t0, t1, t2, t3, t4, t5, t6, t7, t8}
}

func (t T9[T0, T1, T2, T3, T4, T5, T6, T7, T8]) Values() (
	T0, T1, T2, T3, T4, T5, T6, T7, T8) {
	return t.T0, t.T1, t.T2, t.T3, t.T4, t.T5, t.T6, t.T7, t.T8
}

// ------- T10
type T10[T0 any, T1 any, T2 any, T3 any, T4 any, T5 any, T6 any, T7 any, T8 any, T9 any] struct {
	T0 T0
	T1 T1
	T2 T2
	T3 T3
	T4 T4
	T5 T5
	T6 T6
	T7 T7
	T8 T8
	T9 T9
}

func NewT10[T0 any, T1 any, T2 any, T3 any, T4 any, T5 any, T6 any, T7 any, T8 any, T9 any](
	t0 T0, t1 T1, t2 T2, t3 T3, t4 T4, t5 T5, t6 T6, t7 T7, t8 T8, t9 T9) T10[
	T0, T1, T2, T3, T4, T5, T6, T7, T8, T9] {
	return T10[T0, T1, T2, T3, T4, T5, T6, T7, T8, T9]{t0, t1, t2, t3, t4, t5, t6, t7, t8, t9}
}

func (t T10[T0, T1, T2, T3, T4, T5, T6, T7, T8, T9]) Values() (
	T0, T1, T2, T3, T4, T5, T6, T7, T8, T9) {
	return t.T0, t.T1, t.T2, t.T3, t.T4, t.T5, t.T6, t.T7, t.T8, t.T9
}

// ------- T11
type T11[T0 any, T1 any, T2 any, T3 any, T4 any, T5 any, T6 any, T7 any, T8 any, T9 any, T10 any] struct {
	T0  T0
	T1  T1
	T2  T2
	T3  T3
	T4  T4
	T5  T5
	T6  T6
	T7  T7
	T8  T8
	T9  T9
	T10 T10
}

func NewT11[T0 any, T1 any, T2 any, T3 any, T4 any, T5 any, T6 any, T7 any, T8 any, T9 any, T10 any](
	t0 T0, t1 T1, t2 T2, t3 T3, t4 T4, t5 T5, t6 T6, t7 T7, t8 T8, t9 T9, t10 T10) T11[
	T0, T1, T2, T3, T4, T5, T6, T7, T8, T9, T10] {
	return T11[T0, T1, T2, T3, T4, T5, T6, T7, T8, T9, T10]{t0, t1, t2, t3, t4, t5, t6, t7, t8, t9, t10}
}

func (t T11[T0, T1, T2, T3, T4, T5, T6, T7, T8, T9, T10]) Values() (
	T0, T1, T2, T3, T4, T5, T6, T7, T8, T9, T10) {
	return t.T0, t.T1, t.T2, t.T3, t.T4, t.T5, t.T6, t.T7, t.T8, t.T9, t.T10
}

// ------- T12
type T12[T0 any, T1 any, T2 any, T3 any, T4 any, T5 any, T6 any, T7 any, T8 any, T9 any, T10 any, T11 any] struct {
	T0  T0
	T1  T1
	T2  T2
	T3  T3
	T4  T4
	T5  T5
	T6  T6
	T7  T7
	T8  T8
	T9  T9
	T10 T10
	T11 T11
}

func NewT12[T0 any, T1 any, T2 any, T3 any, T4 any, T5 any, T6 any, T7 any, T8 any, T9 any, T10 any, T11 any](
	t0 T0, t1 T1, t2 T2, t3 T3, t4 T4, t5 T5, t6 T6, t7 T7, t8 T8, t9 T9, t10 T10, t11 T11) T12[
	T0, T1, T2, T3, T4, T5, T6, T7, T8, T9, T10, T11] {
	return T12[T0, T1, T2, T3, T4, T5, T6, T7, T8, T9, T10, T11]{t0, t1, t2, t3, t4, t5, t6, t7, t8, t9, t10, t11}
}

func (t T12[T0, T1, T2, T3, T4, T5, T6, T7, T8, T9, T10, T11]) Values() (
	T0, T1, T2, T3, T4, T5, T6, T7, T8, T9, T10, T11) {
	return t.T0, t.T1, t.T2, t.T3, t.T4, t.T5, t.T6, t.T7, t.T8, t.T9, t.T10, t.T11
}
