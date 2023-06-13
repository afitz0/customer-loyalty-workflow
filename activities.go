package starter

type Activities struct{}

func (a *Activities) Activity(greeting string, name string) (string, error) {
	return greeting + " " + name + "!", nil
}
