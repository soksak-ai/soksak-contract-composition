package composition

const ChangeEvent = "composition.changed"

type Change struct {
	PreviousGeneration uint64 `json:"previousGeneration"`
	Generation         uint64 `json:"generation"`
}

func Replace(current, next Settings, expectedGeneration uint64) (Settings, Change, error) {
	if err := ValidateSettings(current); err != nil {
		return Settings{}, Change{}, err
	}
	if err := ValidateSettings(next); err != nil {
		return Settings{}, Change{}, err
	}
	if current.Generation != expectedGeneration {
		return Settings{}, Change{}, ErrGenerationConflict{Expected: expectedGeneration, Actual: current.Generation}
	}
	if next.Generation != current.Generation+1 {
		return Settings{}, Change{}, ErrGenerationStep{Previous: current.Generation, Next: next.Generation}
	}
	return next, Change{PreviousGeneration: current.Generation, Generation: next.Generation}, nil
}

type ErrGenerationConflict struct{ Expected, Actual uint64 }

func (err ErrGenerationConflict) Error() string { return "composition settings generation conflict" }

type ErrGenerationStep struct{ Previous, Next uint64 }

func (err ErrGenerationStep) Error() string {
	return "composition settings generation must advance by one"
}
