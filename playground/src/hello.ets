import { Effect, Schema } from "effect";

export class User extends Schema.Class<User>("User")({
  id: Schema.Number,
  name: Schema.String,
}) {}

export const getUser = (id: number) =>
  Effect.succeed(new User({ id, name: "John" }));

export const program = Effect.gen {
  const user = yield* getUser(1);
  return user.name;
}

