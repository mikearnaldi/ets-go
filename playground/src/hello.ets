import { Effect, Result, Schema } from "effect";

export class User extends Schema.Class<User>("User")({
  id: Schema.Number,
  name: Schema.String,
}) {}

export const getUser = (id: number) =>
  Effect.succeed(new User({ id, name: "John" }));

export const program = Effect.gen {
  const user = run getUser(1);
  return user.name;
}

export const hello = Result.gen {
  const a = run Result.succeed(0)
  return a
}

export const program2 = Effect.gen(function* () {
  const user = yield* getUser(1);
  return user.name;
})