<script lang="ts">
  import Icon from "@iconify/svelte"
  import {
    Clear,
    deSerialiseMove,
    indexToFile,
    indexToRow,
    newBoard,
    parseMove,
    pieceToIcon,
    type Move,
  } from "../library/board"
  import {
    parseBoardState,
    sendMove,
    type ConnectEvent,
    type ConnectViewerEvent,
    type GameEvent,
    type MoveEvent,
  } from "../library/events"
  import { onMount } from "svelte"

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

  let ws = $state<null | WebSocket>(null)

  onMount(() => {
    console.log("creating ws")
    const id = getId()
    ws = new WebSocket(`ws://localhost:3000/api/game/subscribe/${id}`)

    ws.addEventListener("open", event => {
      console.log("open:", event)
    })

    ws.addEventListener("message", event => {
      console.log("received:", event)
      messages.push(event.data)
      if (typeof event.data !== "string") throw new Error("event not string")
      const json: GameEvent = JSON.parse(event.data)
      console.log("received message", json)
      handleEvent(json)
    })
  })

  function handleConnect(event: ConnectEvent) {
    const newBoard = parseBoardState(event)
    colour = event.colour === "w"
    board = newBoard
    board.legalMoves = event.legalMoves?.map(deSerialiseMove) ?? []
  }
  function movePiece(move: Move) {
    const piece = board.state[move.from]
    board.state[move.from] = Clear
    board.state[move.to] = piece
  }
  function handleMove(event: MoveEvent) {
    const move = parseMove(event.move)
    movePiece(move)
    board.legalMoves = event.legalMoves?.map(deSerialiseMove) ?? []
  }
  function handleConnectViewer(event: ConnectViewerEvent) {}
  // function handleEnd(event: EndEvent) {}
  function handleEvent(event: GameEvent) {
    if (event.type === "connect") return handleConnect(event)
    if (event.type === "connectViewer") return handleConnectViewer(event)
    if (event.type === "move") return handleMove(event)
    // if (event.type === "end") return handleEnd(event)
  }

  let board = $state(newBoard())
  let colour = $derived(true)
  let displayPieces = $derived(colour ? board.state.toReversed() : board.state)
  let selected = $state<number | null>(null)

  function fromDisplayIndex(index: number) {
    return colour ? 63 - index : index
  }

  function handleClickPiece(to: number) {
    console.log("clicked", to)
    if (selected === null) {
      selected = to
      return
    }

    const from = selected
    selected = null

    if (board.legalMoves.findIndex(move => move.from === from && move.to === to) === -1) {
      console.log(board.legalMoves)
      console.log(from, to)
      console.warn("not legal move")
      return
    }

    movePiece({ from, to: to })
    const hi = sendMove(from, to)
    console.log("sending", hi)
    ws?.send(JSON.stringify(hi))
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
  {#each displayPieces as piece, index}
    <button
      class={[
        "relative flex h-16 w-16 items-center justify-center",
        {
          "bg-gray-700": isBlack(fromDisplayIndex(index)),
          "bg-gray-200": !isBlack(fromDisplayIndex(index)),
          "border-2 border-red-100": fromDisplayIndex(index) === selected,
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
