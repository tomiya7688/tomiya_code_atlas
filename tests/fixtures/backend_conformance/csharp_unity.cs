using UnityEngine;

namespace Demo.Unity;

public sealed class HeroController : MonoBehaviour
{
    [SerializeField] private float speed;

    public Transform? Target { get; private set; }

    public float Speed { get; init; }

    public void Move(Vector3 direction)
    {
        transform.Translate(direction * Speed);
    }
}

public enum MovementMode
{
    Walk,
    Fly,
}

public readonly struct MoveResult
{
    public Vector3 Position { get; init; }
}
