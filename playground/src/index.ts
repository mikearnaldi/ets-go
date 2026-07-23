import { Effect } from "effect";
import { program, getUser } from "./hello.ets";

const name = Effect.runSync(program);

const user = Effect.runSync(getUser(2));

export const nameLength = name.length + user.id;
