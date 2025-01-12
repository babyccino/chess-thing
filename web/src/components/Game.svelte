<script lang="ts">
  import Icon from "@iconify/svelte"
  import {
    type Piece,
    indexToFile,
    indexToRow,
    initialBoardState,
    pieceToIcon,
    serialiseMove,
  } from "../library/board"

  let messages = $state<any[]>([])

  function getIdFromRoute(): string | null {
    const matches = window.location.pathname.matchAll(/name\/(.+)/g)
    for (const match of matches) {
      return match[1]
    }
    return null
  }
  function getId(): string {
    const routeId = getIdFromRoute()
    if (routeId !== null) return routeId
    const params = new URLSearchParams(window.location.search)
    const id = params.get("gameId")
    if (id === null) throw new Error("no gameId")
    return id
  }

  const id = getId()
  const ws: WebSocket = new WebSocket(`ws://localhost:3000/game/subscribe/${id}`)

  ws.addEventListener("open", event => {
    console.log("open:", event)
  })

  type Event =
    | {
        type: "connect"
        fen: string
        colour: "w" | "b"
        legalMoves: string[]
      }
    | {
        type: "move"
        move: string
        fen: string
        legalMoves: string[]
      }
    | {
        type: "end"
        victor: "w" | "b"
      }

  ws.addEventListener("message", event => {
    console.log("received:", event)
    messages.push(event.data)
    if (typeof event.data !== "string") throw new Error("event not string")
    const json: Event = JSON.parse(event.data)
  })

  let board = $state<Piece[]>(initialBoardState)
  let colour = $state(true)
  let displayBoard = $derived(colour ? board.toReversed() : board)
  let selected = $state<number | null>(null)

  function fromDisplayIndex(index: number) {
    return colour ? 63 - index : index
  }

  function handleClickPiece(square: number) {
    console.log("clicked", square)
    if (selected === null) {
      selected = square
    } else {
      ws.send(serialiseMove(selected, square))
      selected = null
    }
  }

  function isBlack(index: number): boolean {
    const rowOffset = Math.floor(index / 8)
    return Boolean((index + rowOffset) % 2)
  }
</script>

<ul>
  {#each messages as message}
    <li>{message}</li>
  {/each}
</ul>

<div class="grid grid-cols-8">
  {#each displayBoard as piece, index}
    <button
      class={[
        "relative flex h-16 w-16 items-center justify-center",
        {
          "bg-gray-700": isBlack(fromDisplayIndex(index)),
          "bg-gray-200": !isBlack(fromDisplayIndex(index)),
          "border-2 border-red-100": index === selected,
        },
      ]}
      onclick={() => handleClickPiece(fromDisplayIndex(index))}
    >
      <Icon class="h-10 w-10" icon={pieceToIcon(piece)} />
      {#if index % 8 === 0}
        <p class="absolute left-0 top-0 text-xs text-red-200">
          {indexToRow(fromDisplayIndex(index))}
        </p>
      {/if}
      {#if index >= 56}
        <p class="absolute bottom-0 right-[0.1rem] text-xs text-red-200">
          {indexToFile(fromDisplayIndex(index))}
        </p>
      {/if}
    </button>
  {/each}
</div>
