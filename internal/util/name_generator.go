package util

import (
	"math/rand"
	"strings"
	"time"

	"github.com/lucasepe/codename"
)

func GenerateNames() (firstName string, lastName string) {
	firstNames := []string{
		"emma", "liam", "olivia", "noah", "ava", "ethan", "sophia", "mason",
		"isabella", "william", "mia", "james", "charlotte", "benjamin", "amelia",
		"lucas", "harper", "henry", "evelyn", "alexander", "abigail", "michael",
		"emily", "daniel", "elizabeth", "jacob", "mila", "jackson", "ella", "sebastian",
		"avery", "david", "scarlett", "carter", "grace", "jayden", "chloe", "john",
		"victoria", "owen", "riley", "luke", "aria", "gabriel", "lily", "anthony",
		"aurora", "isaac", "layla", "julian",
	}

	lastNames := []string{
		"smith", "johnson", "williams", "brown", "jones", "garcia", "miller", "davis",
		"rodriguez", "martinez", "hernandez", "lopez", "gonzalez", "wilson", "anderson",
		"thomas", "taylor", "moore", "jackson", "martin", "lee", "perez", "thompson",
		"white", "harris", "sanchez", "clark", "ramirez", "lewis", "robinson", "walker",
		"young", "allen", "king", "wright", "scott", "torres", "nguyen", "hill", "flores",
		"green", "adams", "nelson", "baker", "hall", "rivera", "campbell", "mitchell",
		"carter", "roberts", "gomez", "phillips", "evans", "turner", "diaz", "parker",
		"cruz", "edwards", "collins", "reyes", "stewart", "morris", "morales", "murphy",
		"cook", "rogers", "gutierrez", "ortiz", "morgan", "cooper", "peterson", "bailey",
		"reed", "kelly", "howard", "ramos", "kim", "cox", "ward", "richardson", "watson",
		"brooks", "chavez", "wood", "james", "bennett", "gray", "mendoza", "ruiz", "hughes",
		"price", "alvarez", "castillo", "sanders", "patel", "myers", "long", "ross",
		"foster", "jimenez", "powell", "jenkins", "perry", "russell", "sullivan", "bell",
		"coleman", "butler", "henderson", "barnes", "gonzales", "fisher", "vasquez",
		"simmons", "romero", "jordan", "patterson", "alexander", "hamilton", "graham",
		"reynolds", "griffin", "wallace", "moreno", "west", "cole", "hayes", "bryant",
		"herrera", "gibson", "ellis", "tran", "medina", "aguilar", "stevens", "murray",
		"ford", "castro", "marshall", "owens", "harrison", "fernandez", "mcdonald",
		"woods", "washington", "kennedy", "wells", "vargas", "henry", "chen", "freeman",
		"webb", "tucker", "guzman", "burns", "crawford", "olson", "simpson", "porter",
		"hunter", "gordon", "mendez", "silva", "shaw", "snyder", "mason", "dixon",
		"munoz", "hunt", "hicks", "holmes", "palmer", "wagner", "black", "robertson",
		"boyd", "rose", "stone", "salazar", "fox", "warren", "mills", "meyer", "rice",
		"schmidt", "garza", "daniels", "ferguson", "nichols", "stephens", "soto",
		"weaver", "ryan", "gardner", "payne", "grant", "dunn", "kelley", "spencer",
		"hawkins", "arnold", "pierce", "vazquez", "hansen", "peters", "santos", "hart",
		"bradley", "knight", "elliott", "cunningham", "duncan", "armstrong", "hudson",
		"carroll", "lane", "riley", "andrews", "alvarado", "ray", "deleon", "berry",
		"perkins", "hoffman", "johnston", "matthews", "pena", "richards", "contreras",
		"willis", "carpenter", "lawrence", "sandoval", "guerrero", "george", "chapman",
		"rios", "estrada", "ortega", "watkins", "greene", "nunez", "wheeler", "valdez",
	}

	source := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(source)

	// Select a random first name and last name
	firstName = firstNames[rng.Intn(len(firstNames))]
	lastName = lastNames[rng.Intn(len(lastNames))]

	return firstName, lastName
}

func GenerateCatchAllUsername() string {
	rng, err := codename.DefaultRNG()
	if err != nil {
		panic(err)
	}
	name := codename.Generate(rng, 0)
	return strings.ReplaceAll(name, "-", "")
}
