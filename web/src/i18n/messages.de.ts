/**
 * German text for the message keys the Go engine emits (internal/engine, internal/gameservice,
 * internal/api, cmd/wasm). A Go test checks that every key used there exists here and in
 * messages.en.ts. Args: see types.ts; list args named "entries" are joined with commas only.
 */
export const messagesDe = {
  // shared vocabulary used inside engine messages
  // mock backend text (dev only), shown as is
  raw: "{text}",
  "medal.gold": "Gold",
  "medal.silber": "Silber",
  "cardKind.rower": "Ruderer",
  "cardKind.boat": "Boot",
  "cardKind.trainer": "Trainer",
  "race.team": "Teamwertung",

  // phase titles (stations use their data name)
  "title.result": "Ergebnis: {name}",
  "title.gameOver": "Saisonende",
  "title.swap": "Tausch-Event",
  "title.draft": "Kartenauswahl",

  // phase prompts
  "prompt.setup": "Deine Startmannschaft steht: Ruderer, städtische Boote und Trainer. Los geht die Saison!",
  "prompt.training":
    "Verteile genau {points} Trainingspunkte auf deine Ruderer. Motivation lässt sich nicht trainieren.",
  "prompt.camp":
    "Trainingslager: Verteile genau {points} Trainingspunkte auf höchstens {maxRowers} Ruderer. Motivation lässt sich nicht trainieren.",
  "prompt.selection": "Wähle genau {count} Ruderer aus. Jede ausgewählte Person erhält {effect}.",
  "prompt.eventsDealt": "Die Eventkarten wurden verteilt.",
  "prompt.eventCards": "Deine Eventkarten: {cards}",
  "prompt.noEventCards": "Keiner deiner Ruderer kann noch eine Eventkarte erhalten.",
  "prompt.ereignis": "{name}: {description} {details}",
  "prompt.swap":
    "Tausch-Event: Du darfst zwei Eigenschaften von {rower} miteinander tauschen (Motivation ausgenommen) – oder darauf verzichten.",
  "prompt.swapOther": "Ein Mitspieler darf Eigenschaften tauschen.",
  "prompt.regatta":
    "Wähle einen Trainer und melde deine Mannschaften. Jeder Ruderer und jedes Boot darf nur einmal starten.",
  "prompt.regattaWeather":
    "Wetter: {weather}. Wähle einen Trainer und melde deine Mannschaften. Jeder Ruderer und jedes Boot darf nur einmal starten.",
  "prompt.result": "Die Ergebnisse stehen fest. Medaillen und Siegesmotivation wurden vergeben.",
  "prompt.draft": "{player} wählt eine Karte aus dem offenen Angebot.",
  "prompt.gameOver": "Die Saison ist vorbei. Gewonnen hat {winners} mit {points} Medaillenpunkten.",
  "prompt.gameOver_one": "Die Saison ist vorbei. Gewonnen hat {winners} mit {points} Medaillenpunkten.",
  "prompt.gameOver_other": "Die Saison ist vorbei. Gewonnen haben {winners} mit je {points} Medaillenpunkten.",

  // event card effects (reveal prompt)
  "event.gain": "{card}: {rower} erhält {value} {stat}.",
  "event.doubled": "{card}: {stat} von {rower} steigt auf {value}.",
  "event.swap": "{card}: Du darfst zwei Eigenschaften von {rower} tauschen (nicht Motivation).",
  "event.twin": "{card}: {rower} und {partner} sind verbunden. Beim nächsten Training trainieren beide die Punkte des anderen mit.",
  "event.mission": "Geheime Mission: {mission} (Belohnung: {reward})",
  "event.described": "{card}: {rower} – {description}",
  "event.plain": "{card}: {rower}.",

  // Ereignisse
  "ereignis.detail": "{player}: {rower} ({value} {stat}).",
  "ereignis.active": "{name}: {description}",

  // competition notes
  "note.weatherFixed": "{name}: Das Wetter steht fest.",
  "note.krebsChance": "{name}: Krebs-Grundwahrscheinlichkeit {percent} %.",
  "note.materialChance": "{name}: Materialschaden-Wahrscheinlichkeit {percent} %.",
  "note.cityBoats": "{name}: Städtische Boote {value}.",
  "note.guests": "{name}: Gastmannschaften {value} auf jede Eigenschaft.",
  "note.weights": "{name}: {weights}.",
  "note.weight": "{stat} zählt {weight}-fach",

  // log
  "log.setup": "Spielvorbereitung: Jeder Spieler erhält {rowers} Ruderer, städtische Boote und {trainers} Trainer.",
  "log.eventDealt": "{player}: {rower} erhält {card}.",
  "log.twinTraining": "{card}: {rower} und {partner} trainieren gemeinsam.",
  "log.swapSkipped": "{player} verzichtet auf den Tausch bei {rower}.",
  "log.swapped": "{player} tauscht bei {rower} {a} und {b}.",
  "log.missionDone": "Mission erfüllt: {mission} Belohnung: {reward}.",
  "log.missionFailed": "Mission gescheitert: {mission}",
  "log.missionUnfulfilled": "Mission nicht erfüllt: {mission}",
  "log.ereignis": "{name}: {description} {details}",
  "log.trainingPoints": "{name} wirkt: Dieses Training bringt {points} Punkte.",
  "log.sponsor": "Sponsor: {player} darf eine zusätzliche Karte wählen.",
  "log.draftPick": "{player} wählt {card} ({kind}).",
  "log.selection": "{player}: {rowers} erhalten {effect}.",
  "log.weather": "Wetter bei {competition}: {weather}.",
  "log.mood": "Trainerlaune: {entries}.",
  "log.moodEntry": "{trainer} {mood}",
  "log.result": "{competition} – {race}: {entries}",
  "log.resultEntry": "{name} {total}",
  "log.resultEntryMedal": "{medal}: {name} {total}",
  "log.resultIncident": "{entry} ({incident})",

  // errors
  "err.internal": "Interner Fehler.",
  "err.unknownEndpoint": "Unbekannter Endpunkt.",
  "err.badRequestBody": "Ungültige Anfrage: Die Daten konnten nicht gelesen werden.",
  "err.badRequestArgs": "Ungültige Anfrage: Parameter fehlen oder sind keine Zeichenketten.",
  "err.botCount": "Es sind 1 bis {max} Bots möglich.",
  "err.createFailed": "Das Spiel konnte nicht erstellt werden.",
  "err.submitFailed": "Die Entscheidung konnte nicht verarbeitet werden.",
  "err.catalogFailed": "Der Kartenkatalog konnte nicht geladen werden.",
  "err.gameNotFound": "Dieses Spiel gibt es nicht.",
  "err.botFailed": "Ein Bot konnte nicht entscheiden.",
  "err.noToken": "Anmeldung fehlt. Bitte starte ein neues Spiel.",
  "err.badToken": "Ungültige Anmeldung für dieses Spiel.",
  "err.saveFailed": "Das Spiel konnte nicht gespeichert werden.",
  "err.snapshotOld":
    "Dieser Spielstand stammt aus einer älteren Spielversion mit anderen Regeln und kann nicht mehr geladen werden. Bitte starte ein neues Spiel.",
  "err.snapshotVersion": "Unbekanntes Spielstand-Format (Version {version}).",
  "err.snapshotGameID": "Ungültige Spiel-ID im Spielstand.",
  "err.snapshotReplay": "Der Spielstand konnte nicht wiederhergestellt werden.",
  "err.stalePhase": "Diese Entscheidung gehört zu einer bereits abgeschlossenen Phase. Bitte neu laden.",
  "err.gameOver": "Das Spiel ist bereits beendet.",
  "err.unknownPlayer": "Unbekannter Spieler.",
  "err.notYourTurn": "Du bist in dieser Phase nicht an der Reihe.",
  "err.alreadyDecided": "Du hast in dieser Phase bereits entschieden.",
  "err.wrongKind": "Falscher Entscheidungstyp: erwartet „{want}“, erhalten „{got}“.",
  "err.cardNotOffered": "Diese Karte ist nicht im Angebot.",
  "err.noDecision": "In dieser Phase ist keine Entscheidung möglich.",
  "err.swapUnknownStat": "Unbekannte Eigenschaft beim Tausch.",
  "err.swapSameStat": "Wähle zwei verschiedene Eigenschaften zum Tauschen.",
  "err.swapMotivation": "Motivation kann nicht getauscht werden.",
  "err.rowerNotOwned": "Der Ruderer „{rower}“ gehört nicht zu deinem Team.",
  "err.unknownStat": "Unbekannte Eigenschaft „{stat}“.",
  "err.trainMotivation":
    "Motivation kann man nicht trainieren. Sie steigt nur durch Auswahlen, Events, Ereignisse und (befristet) durch Siege.",
  "err.trainPositive": "Trainingspunkte müssen positiv sein.",
  "err.trainTotal": "Du musst genau {points} Trainingspunkte verteilen (verteilt: {total}).",
  "err.campMaxRowers": "Im Trainingslager dürfen höchstens {maxRowers} Ruderer trainieren (gewählt: {count}).",
  "err.selectAge": "{rower} darf hier nicht teilnehmen (falsche Altersklasse).",
  "err.selectTwice": "Jeder Ruderer darf nur einmal ausgewählt werden.",
  "err.selectCount": "Wähle genau {want} Ruderer aus (gewählt: {count}).",
  "err.trainerNotOwned": "Dieser Trainer gehört nicht zu deinem Team.",
  "err.trainerRequired": "Wähle einen Trainer für diesen Wettkampf.",
  "err.unknownRace": "Unbekanntes Rennen „{race}“.",
  "err.raceTwice": "Pro Rennen ist nur eine Meldung erlaubt ({race}).",
  "err.raceSeats": "{race} braucht genau {seats} Ruderer (gemeldet: {count}).",
  "err.rowerTwice": "{rower} darf pro Wettkampf nur einmal starten.",
  "err.rowerAge": "{rower} ist in {race} nicht startberechtigt (Altersklasse).",
  "err.rowerGender": "{rower} ist in {race} nicht startberechtigt (Geschlecht).",
  "err.mixed": "{race}: Mixed-Boote brauchen mindestens ein Mädchen, einen Jungen und zwei Altersgruppen.",
  "err.noOwnBoat": "Für {race} wird kein eigenes Boot gemeldet.",
  "err.boatRequired": "Wähle ein Boot für {race}.",
  "err.boatNotOwned": "Das Boot „{boat}“ gehört nicht zu deinem Team.",
  "err.boatClass": "{race} braucht ein Boot der Klasse {class}, {boat} ist ein {boatClass}.",
  "err.boatTwice": "Das Boot {boat} darf pro Wettkampf nur einmal eingesetzt werden.",
} as const satisfies Record<string, string>;
