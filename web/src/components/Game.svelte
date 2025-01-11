<script lang="ts">
  import Icon from "@iconify/svelte";
  import {
    type Piece,
    indexToFile,
    indexToRow,
    initialBoardState,
    pieceToIcon,
    serialiseMove,
  } from "../library/board";

  let messages = $state<any[]>([]);

  let ws: WebSocket = new WebSocket("ws://localhost:3000/subscribe/hello");

  ws.addEventListener("open", (event) => {
    console.log("open:", event);
  });

  ws.addEventListener("message", (event) => {
    console.log("received:", event);
    messages.push(event.data);
  });

  let board = $state<Piece[]>(initialBoardState);
  let colour = $state(true);
  let displayBoard = $derived(colour ? board.toReversed() : board);
  let selected = $state<number | null>(null);

  function fromDisplayIndex(index: number) {
    return colour ? 63 - index : index;
  }

  function handleClickPiece(square: number) {
    console.log("clicked", square);
    if (selected === null) {
      selected = square;
    } else {
      ws.send(serialiseMove(selected, square));
      selected = null;
    }
  }

  function isBlack(index: number): boolean {
    const rowOffset = Math.floor(index / 8);
    return Boolean((index + rowOffset) % 2);
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
        "w-16 h-16 flex justify-center items-center relative",
        {
          "bg-gray-700": isBlack(fromDisplayIndex(index)),
          "bg-gray-200": !isBlack(fromDisplayIndex(index)),
          "border-2 border-red-100": index === selected,
        },
      ]}
      onclick={() => handleClickPiece(fromDisplayIndex(index))}
    >
      <Icon class="w-10 h-10" icon={pieceToIcon(piece)} />
      {#if index % 8 === 0}
        <p class="absolute top-0 left-0 text-xs text-red-200">
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
