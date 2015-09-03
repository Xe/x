package main

import "fmt"

const url = "https://s3-ap-northeast-1.amazonaws.com/splatoon-data.nintendo.net/stages_info.json"

var mapNames map[string]string

func init() {
	mapNames = map[string]string{
		"eefffc33ee1956b7b70e250d5835aa67be9152d42bc76aa8874987ebdfc19944": "Urchin Underpass",
		"b8067d2839476ec39072e371b4c59fa85454cdb618515af080ca6080772f3264": "Port Mackerel",
		"50c01bca5b3117f4f7893df86d2e2d95435dbb1aae1da6831b8e74838859bc7d": "Saltspray Rig",
		"9a1736540c3fde7e409cb9c7e333441157d88dfe8ce92bc6aafcb9f79c56cb3d": "Blackbelly Skatepark",
		"d7bf0ca4466e980f994ef7b32faeb71d80611a28c5b9feef84a00e3c4c9d7bc1": "Walleye Warehouse",
		"8c69b7c9a81369b5cfd22adbf41c13a8df01969ff3d0e531a8bcb042156bc549": "Arowana Mall",
		"1ac0981d03c18576d9517f40461b66a472168a8f14f6a8714142af9805df7b8c": "Bluefin Depot",
		"c52a7ab7202a576ee18d94be687d97190e90fdcc25fc4b1591c1a8e0c1c299a5": "Kelp Dome",
		"6a6c3a958712adedcceb34f719e220ab0d840d8753e5f51b089d363bd1763c91": "Moray Towers",
		"a54716422edf71ac0b3d20fbb4ba5970a7a78ba304fcf935aaf69254d61ca709": "Flounder Heights",
		"fafe7416d363c7adc8c5c7b0f76586216ba86dcfe3fd89708d672e99bc822adc": "Camp Triggerfish",
	}
}

func englishIfy(s *Stage) string {
	name, ok := mapNames[s.ID]
	if !ok {
		return fmt.Sprintf("Please tell Xena the name of stage id \"%s\" with jpn name %s", s.ID, s.Name)
	}

	return name
}
