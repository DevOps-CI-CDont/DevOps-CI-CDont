import { postTweet } from "@/server/postTweet";
import { useState } from "react";

export function CreateMessage() {
  const [message, setMessage] = useState("");

  return (
    <div className='w-full mt-2 bg-gray-300 shadow-md px-1 py-1 rounded-md'>
      <div className='mx-4 my-2'>
        <span>What&quot;s on your mind?</span>
        <form className='flex flex-row mt-2' onSubmit={(e) => handleSubmit(e)}>
          <input
            type='text'
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            className='px-2 py-1 mr-2 rounded-md w-full border shadown-md'
            placeholder='write here...'
          />
          <button
            className='px-3 py-1 border rounded-md shadow-md bg-blue-500 text-white'
            type='submit'>
            Share
          </button>
        </form>
      </div>
    </div>
  );

  async function handleSubmit(e: any) {
    e.preventDefault();

    try {
      await postTweet(message);
      setMessage("");
    } catch (e) {
      alert("Something went wrong");
    }
  }
}
